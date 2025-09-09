// repository/restaurant_application_repository.go
package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

type RestaurantApplicationRepository struct {
	DB *gorm.DB
}

func NewRestaurantApplicationRepository(db *gorm.DB) *RestaurantApplicationRepository {
	return &RestaurantApplicationRepository{DB: db}
}

// สร้างใบสมัคร
func (r *RestaurantApplicationRepository) CreateApplication(app *entity.RestaurantApplication) error {
	return r.DB.Create(app).Error
}

// หาใบสมัครทั้งหมดตามสถานะ
func (r *RestaurantApplicationRepository) FindByStatus(status string) ([]entity.RestaurantApplication, error) {
	var apps []entity.RestaurantApplication
	err := r.DB.
			Preload("OwnerUser").
			Preload("RestaurantCategory").
			Where("status = ?", status).
			Order("id DESC").
			Find(&apps).Error
	return apps, err
}

// หาใบสมัครตาม ID
func (r *RestaurantApplicationRepository) FindByID(id uint) (*entity.RestaurantApplication, error) {
	var app entity.RestaurantApplication
	if err := r.DB.
			Preload("OwnerUser").
			Preload("RestaurantCategory").
			First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// หา User ตาม ID
func (r *RestaurantApplicationRepository) FindUserByID(id uint) (*entity.User, error) {
	var u entity.User
	if err := r.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// สร้างร้านจากใบสมัคร + อัปเดตสถานะ
func (r *RestaurantApplicationRepository) CreateRestaurantAndApprove(app *entity.RestaurantApplication, rest *entity.Restaurant, now time.Time) error {
	tx := r.DB.Begin()

	if err := tx.Create(rest).Error; err != nil {
			tx.Rollback()
			return err
	}

	// อัปเกรด role user → owner
	if err := tx.Model(&entity.User{}).Where("id = ?", app.OwnerUserID).
			Where("role = '' OR role = 'customer'").
			Update("role", "owner").Error; err != nil {
			tx.Rollback()
			return err
	}

	// อัปเดต application
	app.Status = "approved"
	app.ReviewedAt = &now
	if err := tx.Save(app).Error; err != nil {
			tx.Rollback()
			return err
	}

	return tx.Commit().Error
}

// Reject Application
func (r *RestaurantApplicationRepository) RejectApplication(app *entity.RestaurantApplication, reason string, adminID *uint, now time.Time) error {
	app.Status = "rejected"
	app.ReviewedAt = &now
	app.AdminID = adminID
	app.RejectReason = &reason
	return r.DB.Save(app).Error
}
