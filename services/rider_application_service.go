package services

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"time"
)

type RiderApplicationService struct{ Repo *repository.RiderApplicationRepository }

func NewRiderApplicationService(repo *repository.RiderApplicationRepository) *RiderApplicationService {
	return &RiderApplicationService{Repo: repo}
}

func (s *RiderApplicationService) List(status string) ([]entity.RiderApplication, error) {
	if status == "" { status = "pending" }
	return s.Repo.FindByStatus(status)
}

func (s *RiderApplicationService) ListMine(userID uint, status string) ([]entity.RiderApplication, error) {
	return s.Repo.FindByUser(userID, status)
}

func (s *RiderApplicationService) Apply(app *entity.RiderApplication) (uint, error) {
	app.Status = "pending"
	if err := s.Repo.CreateApplication(app); err != nil {
		return 0, err
	}
	return app.ID, nil
}

type RiderApproveReq struct {
	AdminID *uint  `json:"adminId,omitempty"`
	NewRole *string `json:"newRole,omitempty"` // เผื่ออยากเซ็ตเป็นค่าอื่น นอกเหนือจาก default "rider"
}

func (s *RiderApplicationService) Approve(appID uint, req RiderApproveReq) (*entity.Rider, *entity.User, error) {
	app, err := s.Repo.FindByID(appID)
	if err != nil { return nil, nil, err }
	if app.Status != "pending" {
		return nil, nil, errors.New("application is not pending")
	}

	// map จากใบสมัคร → Rider จริง
	r := entity.Rider{
		UserID:       app.UserID,
		VehiclePlate: app.VehiclePlate,
		License:      app.License,
		// เนื่องจากใน application เก็บรูป (base64) ไว้ที่ DriveCar (drive_car_picture)
		// โครงนี้จะใช้ตรรกะง่าย ๆ: ถ้ามีรูปแนบ → DriveCar = true (ปรับตามนโยบายของคุณได้)
		DriveCar:     app.DriveCar != "",
		RiderStatusID: 1, // ค่าเริ่มต้น เช่น "available" หรือปรับตาม seed ของคุณ
		AdminID:       req.AdminID,
	}

	now := time.Now()
	if err := s.Repo.CreateRiderAndApprove(app, &r, now); err != nil {
		return nil, nil, err
	}

	owner, err := s.Repo.FindUserByID(app.UserID)
	if err != nil {
		return &r, nil, err
	}

	// ถ้ากำหนด NewRole มาและต้องการบังคับ role เป็นค่านั้น ให้ทำที่ชั้น Repo เพิ่มเติมได้
	if req.NewRole != nil && *req.NewRole != "" && owner.Role != *req.NewRole {
		owner.Role = *req.NewRole
		// ปล. ถ้าต้อง persist ตรงนี้จริง ๆ แนะนำเพิ่มเมธอดอัปเดต role ลง DB ที่ Repo
	}

	return &r, owner, nil
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
