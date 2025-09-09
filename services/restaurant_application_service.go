// services/restaurant_application_service.go
package services

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"time"
)

type RestaurantApplicationService struct {
	Repo *repository.RestaurantApplicationRepository
}

func NewRestaurantApplicationService(repo *repository.RestaurantApplicationRepository) *RestaurantApplicationService {
	return &RestaurantApplicationService{Repo: repo}
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

    statusID := req.RestaurantStatusID
    if statusID == 0 {
        statusID = 1 // default Open
    }

    now := time.Now()
    rest := entity.Restaurant{
        Name:                 app.Name,
        Address:              app.Address,
        Description:          app.Description,
        Picture:              app.Picture,      // ✅ base64
        OpeningTime:          app.OpeningTime,  // ✅ เวลาเปิด
        ClosingTime:          app.ClosingTime,  // ✅ เวลาปิด
        RestaurantCategoryID: app.RestaurantCategoryID,
        RestaurantStatusID:   statusID,
        UserID:               app.OwnerUserID,
        AdminID:              req.AdminID,
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
