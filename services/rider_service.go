// services/rider_service.go
package services

import (
	"backend/repository"
	"backend/entity" // << สำคัญ: สำหรับ User/Rider entity
	"errors"
	"sync"
	"time"

	"gorm.io/gorm"
)

type RiderService struct {
	DB        *gorm.DB
	RiderRepo *repository.RiderRepository
	WorkRepo  *repository.RiderWorkRepository
	OrderRepo *repository.OrderRepository

	// lazy-loaded status IDs (resolve ครั้งเดียว)
	once              sync.Once
	StatusOfflineID   uint
	StatusOnlineID    uint
	StatusAssignedID  uint
	StatusCompletedID uint

	OrderPreparingID  uint
	OrderDeliveringID uint
	OrderCompletedID  uint

	initErr error
}

func NewRiderService(
	db *gorm.DB,
	riderRepo *repository.RiderRepository,
	workRepo *repository.RiderWorkRepository,
	orderRepo *repository.OrderRepository,
) *RiderService {
	return &RiderService{
		DB: db, RiderRepo: riderRepo, WorkRepo: workRepo, OrderRepo: orderRepo,
	}
}

// initIDs: ดึง ID ของ RiderStatus / OrderStatus จาก "ชื่อ" เพียงครั้งเดียว
func (s *RiderService) initIDs() error {
	s.once.Do(func() {
		var err error

		// RiderStatus
		if s.StatusOfflineID, err = s.RiderRepo.GetStatusIDByName("OFFLINE"); err != nil { s.initErr = err; return }
		if s.StatusOnlineID, err = s.RiderRepo.GetStatusIDByName("ONLINE"); err != nil { s.initErr = err; return }
		if s.StatusAssignedID, err = s.RiderRepo.GetStatusIDByName("ASSIGNED"); err != nil { s.initErr = err; return }
		if s.StatusCompletedID, err = s.RiderRepo.GetStatusIDByName("COMPLETED"); err != nil { s.initErr = err; return }

		// OrderStatus (ใช้ helper เดิมใน OrderRepository)
		if s.OrderPreparingID, err = s.OrderRepo.GetStatusIDByName("Preparing"); err != nil { s.initErr = err; return }
		if s.OrderDeliveringID, err = s.OrderRepo.GetStatusIDByName("Delivering"); err != nil { s.initErr = err; return }
		if s.OrderCompletedID, err = s.OrderRepo.GetStatusIDByName("Completed"); err != nil { s.initErr = err; return }
	})
	return s.initErr
}

// 1) Rider ONLINE/OFFLINE
func (s *RiderService) SetAvailability(userID uint, online bool) error {
	if err := s.initIDs(); err != nil { return err }

	r, err := s.RiderRepo.GetByUserID(userID)
	if err != nil { return err }

	// ขอ OFFLINE แต่ยังมีงานค้างอยู่ → ไม่ให้
	if !online {
		has, err := s.RiderRepo.HasActiveWork(r.ID)
		if err != nil { return err }
		if has {
			return errors.New("cannot go OFFLINE while having active work")
		}
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		target := s.StatusOfflineID
		if online { target = s.StatusOnlineID }
		return s.RiderRepo.UpdateStatus(tx, r.ID, target)
	})
}

// 2) Rider รับงาน: ONLINE → ASSIGNED + Order Preparing → Delivering
func (s *RiderService) AcceptWork(userID, orderID uint) error {
	if err := s.initIDs(); err != nil { return err }

	r, err := s.RiderRepo.GetByUserID(userID)
	if err != nil { return err }

	if r.RiderStatusID != s.StatusOnlineID {
		return errors.New("rider is not ONLINE")
	}
	if ok, err := s.RiderRepo.HasActiveWork(r.ID); err != nil {
		return err
	} else if ok {
		return errors.New("rider already has an active work")
	}
	// กันซ้ำ: ออเดอร์นี้ถูกใครรับไปแล้วหรือยัง
	if ok, err := s.WorkRepo.HasWorkForOrder(orderID); err != nil {
		return err
	} else if ok {
		return errors.New("order already assigned")
	}

	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// create RiderWork
		if err := s.WorkRepo.CreateAssign(tx, r.ID, orderID, now); err != nil {
			return err
		}
		// rider: ONLINE → ASSIGNED
		if err := s.RiderRepo.UpdateStatus(tx, r.ID, s.StatusAssignedID); err != nil {
			return err
		}
		// order: Preparing → Delivering
		ok, err := s.OrderRepo.UpdateStatusFromTo(tx, orderID, s.OrderPreparingID, s.OrderDeliveringID)
		if err != nil { return err }
		if !ok { return errors.New("order not in preparing state") }
		return nil
	})
}

// 3) ส่งสำเร็จ: ASSIGNED → COMPLETED → ONLINE + Order Delivering → Completed
func (s *RiderService) CompleteWork(userID, orderID uint) error {
	if err := s.initIDs(); err != nil { return err }

	r, err := s.RiderRepo.GetByUserID(userID)
	if err != nil { return err }
	if r.RiderStatusID != s.StatusAssignedID {
		return errors.New("rider is not on an assigned work")
	}

	now := time.Now()
	return s.DB.Transaction(func(tx *gorm.DB) error {
		// ปิด RiderWork
		if err := s.WorkRepo.FinishWork(tx, r.ID, orderID, now); err != nil {
			return err
		}
		// order: Delivering → Completed
		ok, err := s.OrderRepo.UpdateStatusFromTo(tx, orderID, s.OrderDeliveringID, s.OrderCompletedID)
		if err != nil { return err }
		if !ok { return errors.New("order not in delivering state") }

		// set COMPLETED แล้วเด้งกลับ ONLINE (ให้ UI รู้ว่าเพิ่ง complete)
		if err := s.RiderRepo.UpdateStatus(tx, r.ID, s.StatusCompletedID); err != nil {
			return err
		}
		if err := s.RiderRepo.UpdateStatus(tx, r.ID, s.StatusOnlineID); err != nil {
			return err
		}
		return nil
	})
}

func (s *RiderService) ListAvailable() ([]repository.AvailableOrderRow, error) {
	if err := s.initIDs(); err != nil { return nil, err }
	return s.WorkRepo.ListAvailable(s.OrderPreparingID, 50)
}

func (s *RiderService) GetStatus(userID uint) (map[string]any, error) {
	if err := s.initIDs(); err != nil {
		return nil, err
	}
	r, err := s.RiderRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"status":    r.RiderStatus.StatusName, // เช่น "ONLINE"
		"isWorking": r.RiderStatusID != s.StatusOfflineID,
	}, nil
}

func (s *RiderService) GetCurrentWork(userID uint) (*repository.AvailableOrderRow, error) {
	if err := s.initIDs(); err != nil {
		return nil, err
	}
	r, err := s.RiderRepo.GetByUserID(userID)
	if err != nil {
		return nil, err
	}
	return s.WorkRepo.FindActiveWork(r.ID)
}

// ===================== โปรไฟล์ Rider (ใหม่) =====================

// ---- DTO สำหรับโปรไฟล์ ----
type ProfileSnapshot struct {
	User  entity.User
	Rider *entity.Rider
}

type UpsertRiderProfileInput struct {
	NationalID   string
	VehiclePlate string
	Zone         string
	License      string
	// nil = ไม่แตะรูป, "" = ลบรูป, "xxxxx" = base64 (strip แล้ว)
	DriveCardB64 *string
}

// ดึง snapshot โปรไฟล์ (users + riders) ของ user ที่ล็อกอิน
func (s *RiderService) GetProfileSnapshot(userID uint) (*ProfileSnapshot, error) {
	var u entity.User
	if err := s.DB.First(&u, userID).Error; err != nil {
		return nil, err
	}

	var r entity.Rider
	if err := s.DB.Where("user_id = ?", userID).First(&r).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &ProfileSnapshot{User: u, Rider: nil}, nil
		}
		return nil, err
	}
	return &ProfileSnapshot{User: u, Rider: &r}, nil
}

// สร้าง/อัปเดต โปรไฟล์ rider ของ user นี้
func (s *RiderService) UpsertRiderProfile(userID uint, in UpsertRiderProfileInput) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var r entity.Rider
		err := tx.Where("user_id = ?", userID).First(&r).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// create ใหม่
				r = entity.Rider{
					UserID:       userID,
					NationalID:   in.NationalID,
					VehiclePlate: in.VehiclePlate,
					Zone:         in.Zone,
					License:      in.License,
				}
				if in.DriveCardB64 != nil {
					r.DriveCard = *in.DriveCardB64 // ""=ลบ, "xxxxx"=รูปใหม่
				}
				return tx.Create(&r).Error
			}
			return err
		}

		// update ของเดิม
		r.NationalID = in.NationalID
		r.VehiclePlate = in.VehiclePlate
		r.Zone = in.Zone
		r.License = in.License
		if in.DriveCardB64 != nil {
			r.DriveCard = *in.DriveCardB64 // ""=ลบ, "xxxxx"=รูปใหม่
		}
		return tx.Save(&r).Error
	})
}
