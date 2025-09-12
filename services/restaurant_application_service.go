// services/restaurant_application_service.go
package services

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"strings"
	"time"
)

type RestaurantApplicationService struct {
	Repo *repository.RestaurantApplicationRepository
}

func NewRestaurantApplicationService(repo *repository.RestaurantApplicationRepository) *RestaurantApplicationService {
	return &RestaurantApplicationService{Repo: repo}
}

// ===== helpers =====
func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}
func isValidPromptPay(d string) bool {
	// 10 = เบอร์มือถือ, 13 = เลขบัตรประชาชน
	return len(d) == 10 || len(d) == 13
}

// List Applications by status
func (s *RestaurantApplicationService) List(status string) ([]entity.RestaurantApplication, error) {
	if status == "" {
		status = "pending"
	}
	return s.Repo.FindByStatus(status)
}

func (s *RestaurantApplicationService) Apply(app *entity.RestaurantApplication, base64Img string) (uint, error) {
	if base64Img != "" {
		app.Picture = base64Img // เก็บ base64 ลงตรง ๆ
	}
	// sanitize PromptPay ตั้งแต่สมัคร (ถ้ามีส่งมา)
	if app.PromptPay != "" {
		app.PromptPay = digitsOnly(app.PromptPay)
	}

	app.Status = "pending"
	if err := s.Repo.CreateApplication(app); err != nil {
		return 0, err
	}
	return app.ID, nil
}

// Approve Application
type ApproveReq struct {
	RestaurantStatusID uint  `json:"restaurantStatusId"`
	AdminID            *uint `json:"adminId,omitempty"`
}

func (s *RestaurantApplicationService) Approve(appID uint, req ApproveReq) (*entity.Restaurant, *entity.User, error) {
	app, err := s.Repo.FindByID(appID)
	if err != nil {
		return nil, nil, err
	}
	if app.Status != "pending" {
		return nil, nil, errors.New("application is not pending")
	}

	// sanitize + validate promptpay จากใบสมัคร
	cleanPP := digitsOnly(app.PromptPay)
	if !isValidPromptPay(cleanPP) {
		return nil, nil, errors.New("invalid promptpay in application (must be 10 or 13 digits)")
	}

	statusID := req.RestaurantStatusID
	if statusID == 0 {
		statusID = 1 // default Open
	}

	now := time.Now()
	rest := entity.Restaurant{
		Name:                 app.Name,
		Address:              app.Address,
		Description:          app.Description,
		Picture:              app.Picture,     // base64 จากใบสมัคร
		OpeningTime:          app.OpeningTime, // เวลาเปิด
		ClosingTime:          app.ClosingTime, // เวลาปิด
		RestaurantCategoryID: app.RestaurantCategoryID,
		RestaurantStatusID:   statusID,
		UserID:               app.OwnerUserID,
		AdminID:              req.AdminID,
		PromptPay:            cleanPP, //PromptPay (เก็บเป็นเลขล้วน จะเป็นเบอร์หรือเลขบัตรก็ได้)
	}

	if err := s.Repo.CreateRestaurantAndApprove(app, &rest, now); err != nil {
		return nil, nil, err
	}

	owner, err := s.Repo.FindUserByID(app.OwnerUserID)
	if err != nil {
		return &rest, nil, err
	}

	return &rest, owner, nil
}

// Reject Application
func (s *RestaurantApplicationService) Reject(appID uint, reason string, adminID *uint) error {
	app, err := s.Repo.FindByID(appID)
	if err != nil {
		return err
	}
	if app.Status != "pending" {
		return errors.New("cannot reject application with status " + app.Status)
	}

	now := time.Now()
	return s.Repo.RejectApplication(app, reason, adminID, now)
}
