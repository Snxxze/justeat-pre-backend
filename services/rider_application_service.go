package services

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"time"
)

type RiderApplicationService struct{
	Repo      *repository.RiderApplicationRepository
	RiderRepo *repository.RiderRepository // << เพิ่ม
}

func NewRiderApplicationService(
	repo *repository.RiderApplicationRepository,
	riderRepo *repository.RiderRepository, // << เพิ่ม
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

func (s *RiderApplicationService) Apply(app *entity.RiderApplication) (uint, error) {
	app.Status = "pending"
	if err := s.Repo.CreateApplication(app); err != nil {
		return 0, err
	}
	return app.ID, nil
}

type RiderApproveReq struct {
	AdminID *uint   `json:"adminId,omitempty"`
	NewRole *string `json:"newRole,omitempty"`
}

func (s *RiderApplicationService) Approve(appID uint, req RiderApproveReq) (*entity.Rider, *entity.User, error) {
	app, err := s.Repo.FindByID(appID)
	if err != nil { return nil, nil, err }
	if app.Status != "pending" {
		return nil, nil, errors.New("application is not pending")
	}

	// ดึง id ของสถานะเริ่มต้นจากตาราง ไม่ฮาร์ดโค้ด
	onlineID, err := s.RiderRepo.GetStatusIDByName("ONLINE")
	if err != nil || onlineID == 0 {
		// fallback: ถ้าไม่พบ ONLINE ให้ลอง OFFLINE
		onlineID, _ = s.RiderRepo.GetStatusIDByName("OFFLINE")
	}

	r := entity.Rider{
		UserID:        app.UserID,
		VehiclePlate:  app.VehiclePlate,
		License:       app.License,
		DriveCar:      app.DriveCar != "", // ถ้ามีรูป/หลักฐานให้ถือว่า driveCar = true
		RiderStatusID: onlineID,           // ใช้ id ที่ lookup ได้
		AdminID:       req.AdminID,
	}

	now := time.Now()
	if err := s.Repo.CreateRiderAndApprove(app, &r, now); err != nil {
		return nil, nil, err
	}

	owner, err := s.Repo.FindUserByID(app.UserID)
	if err != nil { return &r, nil, err }

	if req.NewRole != nil && *req.NewRole != "" && owner.Role != *req.NewRole {
		owner.Role = *req.NewRole
		// ถ้าจะ persist ตรงนี้จริง ให้เพิ่มเมธอดอัปเดต role ใน Repo
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
