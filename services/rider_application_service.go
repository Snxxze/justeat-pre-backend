package services

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"backend/entity"
	"backend/repository"
)

// ส่งให้ Controller เช็คเพื่อคืน 409
var ErrAlreadyApplied = errors.New("already_applied")

type RiderApplicationService struct{
	Repo      *repository.RiderApplicationRepository
	RiderRepo *repository.RiderRepository
}

func NewRiderApplicationService(
	repo *repository.RiderApplicationRepository,
	riderRepo *repository.RiderRepository,
) *RiderApplicationService {
	return &RiderApplicationService{Repo: repo, RiderRepo: riderRepo}
}

func (s *RiderApplicationService) List(status string) ([]entity.RiderApplication, error) {
	if status == "" { status = "pending" }
	return s.Repo.FindByStatus(status)
}

func (s *RiderApplicationService) ListMine(userID uint, status string) ([]entity.RiderApplication, error) {
	return s.Repo.FindByUser(userID, status)
}

// อนุญาตให้สมัครใหม่ได้เมื่อใบสมัครก่อนหน้า "rejected"
// กันซ้ำเฉพาะกรณีมีของเดิมสถานะ 'pending' หรือ 'approved'
func (s *RiderApplicationService) Apply(app *entity.RiderApplication) (uint, error) {
	var cnt int64
	if err := s.Repo.DB.
		Model(&entity.RiderApplication{}).
		Where("user_id = ? AND status IN ('pending','approved')", app.UserID).
		Count(&cnt).Error; err != nil {
		return 0, err
	}
	if cnt > 0 {
		return 0, ErrAlreadyApplied
	}

	app.Status = "pending"
	if err := s.Repo.CreateApplication(app); err != nil {
		return 0, err
	}
	return app.ID, nil
}

type RiderApproveReq struct {
	AdminID *uint   `json:"adminId,omitempty"`
	NewRole *string `json:"newRole,omitempty"` // ถ้าไม่กำหนด จะใช้ default logic ด้านล่าง
}

// อนุมัติใบสมัคร Rider
func (s *RiderApplicationService) Approve(appID uint, req RiderApproveReq) (*entity.Rider, *entity.User, error) {
	var rider entity.Rider
	var owner entity.User

	err := s.Repo.DB.Transaction(func(tx *gorm.DB) error {
		// 1) โหลดใบสมัคร + เจ้าของ
		var app entity.RiderApplication
		if err := tx.Preload("User").First(&app, appID).Error; err != nil {
			return err
		}
		if app.Status == "approved" {
			return fmt.Errorf("application already approved")
		}
		owner = app.User

		// 2) หา/สร้าง Rider และคัดลอก drive_card (base64)
		var r entity.Rider
		err := tx.Where("user_id = ?", app.UserID).First(&r).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			r = entity.Rider{
				UserID:       app.UserID,
				VehiclePlate: app.VehiclePlate,
				License:      app.License,
				DriveCard:    app.DriveCar,
			}
			if err := tx.Create(&r).Error; err != nil {
				return err
			}
		case err == nil:
			if err := tx.Model(&r).UpdateColumn("drive_card", app.DriveCar).Error; err != nil {
				return err
			}
		default:
			return err
		}
		rider = r

		// 3) อัปเดตสถานะใบสมัคร
		now := time.Now()
		app.Status = "approved"
		app.ReviewedAt = &now
		if req.AdminID != nil {
			app.AdminID = req.AdminID
		}
		if err := tx.Save(&app).Error; err != nil {
			return err
		}

		// 4) อัปเดต role ของผู้ใช้แบบ "ไม่ทับสิทธิ์เดิม"
		//    - ถ้า role ปัจจุบันว่างหรือ "customer" => ตั้งเป็น "rider" (หรือใช้ req.NewRole ถ้าส่งมา)
		//    - ถ้าเป็น "partner"/"admin"/อื่น ๆ => คงเดิม ไม่ทับ
		current := strings.ToLower(strings.TrimSpace(owner.Role))
		targetRole := current
		if current == "" || current == "customer" {
			targetRole = "rider"
			if req.NewRole != nil && *req.NewRole != "" {
				targetRole = *req.NewRole
			}
		}

		if targetRole != current {
			if err := tx.Model(&owner).Update("role", targetRole).Error; err != nil {
				return err
			}
			owner.Role = targetRole
		}

		return nil
	})

	if err != nil { return nil, nil, err }
	return &rider, &owner, nil
}

func (s *RiderApplicationService) Reject(appID uint, reason string, adminID *uint) error {
	app, err := s.Repo.FindByID(appID)
	if err != nil { return err }
	if app.Status != "pending" {
		return errors.New("cannot reject application with status " + app.Status)
	}
	now := time.Now()
	return s.Repo.RejectApplication(app, reason, adminID, now)
}
