// repository/rider_application_repository.go
package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

type RiderApplicationRepository struct{ DB *gorm.DB }

func NewRiderApplicationRepository(db *gorm.DB) *RiderApplicationRepository {
	return &RiderApplicationRepository{DB: db}
}

// ผู้ใช้ยื่นสมัคร
func (r *RiderApplicationRepository) CreateApplication(app *entity.RiderApplication) error {
	return r.DB.Create(app).Error
}

// แอดมินดูรายการตามสถานะ
func (r *RiderApplicationRepository) FindByStatus(status string) ([]entity.RiderApplication, error) {
	var apps []entity.RiderApplication
	err := r.DB.
		Preload("User").
		Where("status = ?", status).
		Order("id DESC").
		Find(&apps).Error
	return apps, err
}

// ผู้ใช้ดูรายการของตัวเอง (option: กรองสถานะ)
func (r *RiderApplicationRepository) FindByUser(userID uint, status string) ([]entity.RiderApplication, error) {
	var apps []entity.RiderApplication
	q := r.DB.Preload("User").Where("user_id = ?", userID).Order("id DESC")
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Find(&apps).Error
	return apps, err
}

// หาใบสมัครตาม ID
func (r *RiderApplicationRepository) FindByID(id uint) (*entity.RiderApplication, error) {
	var app entity.RiderApplication
	if err := r.DB.
		Preload("User").
		First(&app, id).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

// หา User ตาม ID
func (r *RiderApplicationRepository) FindUserByID(id uint) (*entity.User, error) {
	var u entity.User
	if err := r.DB.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

// สร้าง Rider จริง + อัปเดตสถานะใบสมัครเป็น approved + upgrade role
func (r *RiderApplicationRepository) CreateRiderAndApprove(app *entity.RiderApplication, rd *entity.Rider, now time.Time) error {
	tx := r.DB.Begin()

	if err := tx.Create(rd).Error; err != nil {
		tx.Rollback()
		return err
	}

	// อัปเกรด role เป็น rider กรณีเดิมเป็นค่าว่างหรือ customer
	if err := tx.Model(&entity.User{}).Where("id = ?", app.UserID).
		Where("role = '' OR role = 'customer' OR role IS NULL").
		Update("role", "rider").Error; err != nil {
		tx.Rollback()
		return err
	}

	app.Status = "approved"
	app.ReviewedAt = &now
	if err := tx.Save(app).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

// ปฏิเสธใบสมัคร
func (r *RiderApplicationRepository) RejectApplication(app *entity.RiderApplication, reason string, adminID *uint, now time.Time) error {
	app.Status = "rejected"
	app.ReviewedAt = &now
	app.AdminID = adminID
	app.RejectReason = &reason
	return r.DB.Save(app).Error
}
