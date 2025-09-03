package repository

import (
	"backend/entity"

	"gorm.io/gorm"
)

// UserRepository รับผิดชอบการคุยกับตาราง users ใน DB เท่านั้น
type UserRepository struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// หาผู้ใช้จาก email
func (r *UserRepository) FindByEmail(email string) (*entity.User, error) {
	var user entity.User
	if err := r.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// นับจำนวน user ที่มี email ซ้ำ
func (r *UserRepository) CountByEmail(email string) (int64, error) {
	var count int64
	if err := r.DB.Model(&entity.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// สร้าง user ใหม่
func (r *UserRepository) Create(user *entity.User) error {
	return r.DB.Create(user).Error
}

// อัปเดต user
func (r *UserRepository) Update(userID uint, updates map[string]any) error {
	return r.DB.Model(&entity.User{}).Where("id = ?", userID).Updates(updates).Error
}

// โหลด user ตาม ID
func (r *UserRepository) FindByID(id uint) (*entity.User, error) {
	var user entity.User
	if err := r.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// บันทึก avatar ลง DB
func (r *UserRepository) SaveAvatar(userID uint, data []byte, contentType string) error {
	return r.DB.Model(&entity.User{}).Where("id = ?", userID).
		Updates(map[string]any{
			"avatar":      data,
			"avatar_type": contentType,
			"avatar_size": len(data),
		}).Error
}

// ดึง avatar ออกมา
func (r *UserRepository) GetAvatar(userID uint) (*entity.User, error) {
	var u entity.User
	if err := r.DB.Select("avatar, avatar_type, avatar_size").First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}