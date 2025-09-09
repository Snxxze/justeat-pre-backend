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

// ✅ บันทึก Avatar (Base64)
func (r *UserRepository) SaveAvatarBase64(userID uint, b64 string) error {
	return r.DB.Model(&entity.User{}).
		Where("id = ?", userID).
		Update("avatar_base64", b64).Error
}

// ✅ ดึง Avatar (Base64)
func (r *UserRepository) FindAvatarBase64(userID uint) (string, error) {
	var u entity.User
	if err := r.DB.Select("avatar_base64").First(&u, userID).Error; err != nil {
		return "", err
	}
	return u.AvatarBase64, nil
}

// ✅ ลบ Avatar (Base64)
func (r *UserRepository) DeleteAvatarBase64(userID uint) error {
	return r.DB.Model(&entity.User{}).
		Where("id = ?", userID).
		Update("avatar_base64", "").Error
}

func (r *UserRepository) FindWithRestaurant(id uint) (*entity.User, *entity.Restaurant, error) {
	var user entity.User
	if err := r.DB.First(&user, id).Error; err != nil {
			return nil, nil, err
	}

	var restaurant entity.Restaurant
	if user.Role == "owner" {
			if err := r.DB.Where("user_id = ?", id).First(&restaurant).Error; err != nil {
					return &user, nil, nil // owner ที่ยังไม่มีร้าน
			}
	}

	return &user, &restaurant, nil
}

func (r *UserRepository) FindRestaurantByUserID(userID uint) (*entity.Restaurant, error) {
    var restaurant entity.Restaurant
    if err := r.DB.
        Where("user_id = ?", userID).
        First(&restaurant).Error; err != nil {
        return nil, err
    }
    return &restaurant, nil
}