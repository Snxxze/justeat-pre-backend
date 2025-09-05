// services/UserPromotionService.go
package services

import (
	"errors"

	"backend/entity"
	"gorm.io/gorm"
)

// error กลางที่ controller ใช้ตรวจชนิด
var (
	ErrAlreadySaved          = errors.New("already saved")
	ErrPromotionNotFound     = errors.New("promotion not found")
	ErrUserPromotionNotFound = errors.New("user promotion not found")
	ErrAlreadyUsed           = errors.New("already used")
)

type UserPromotionService struct {
	DB *gorm.DB
}

func NewUserPromotionService(db *gorm.DB) *UserPromotionService {
	return &UserPromotionService{DB: db}
}

// SavePromotion: บันทึกโปรให้ผู้ใช้ (กันเก็บซ้ำ)
// - ตรวจว่า promotion มีอยู่จริง
// - สร้างแถว user_promotions ด้วยคีย์ unique (user_id + promotion_id)
// - ถ้าชน unique ให้คืน ErrAlreadySaved (กัน race จากการกดซ้ำพร้อมกัน)
func (s *UserPromotionService) SavePromotion(userID, promoID uint) error {
	// 1) ตรวจว่ามี promotion นี้จริงไหม (และไม่ถูก soft-delete)
	var cnt int64
	if err := s.DB.Model(&entity.Promotion{}).Where("id = ?", promoID).Count(&cnt).Error; err != nil {
		return err
	}
	if cnt == 0 {
		return ErrPromotionNotFound
	}

	// 2) พยายามสร้างแถวใหม่ (พึ่งพา unique index: user_id + promotion_id)
	up := entity.UserPromotion{
		UserID:      userID,
		PromotionID: promoID,
		IsUsed:      false,
	}

	if err := s.DB.Create(&up).Error; err != nil {
		// ถ้า DB ตอบ duplicated key (รองรับได้กับหลาย driver ผ่าน gorm.ErrDuplicatedKey)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrAlreadySaved
		}
		// (สำรอง) สำหรับบางไดรเวอร์อาจไม่ map เป็น gorm.ErrDuplicatedKey
		// ผู้ใช้ส่วนใหญ่ไม่จำเป็น แต่ถ้าอยากละเอียดขึ้น สามารถ parse error string
		// สำหรับ sqlite: "UNIQUE constraint failed: user_promotions.user_id, user_promotions.promotion_id"
		return err
	}

	return nil
}

// UsePromotion: ทำเครื่องหมายว่าโปรนี้ถูกใช้แล้วโดยผู้ใช้คนนี้
// - ต้องมีแถว user_promotions อยู่ก่อน
// - ถ้าใช้แล้วซ้ำ => ErrAlreadyUsed
func (s *UserPromotionService) UsePromotion(userID, promoID uint) error {
	var up entity.UserPromotion
	if err := s.DB.Where("user_id = ? AND promotion_id = ?", userID, promoID).
		First(&up).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserPromotionNotFound
		}
		return err
	}

	if up.IsUsed {
		return ErrAlreadyUsed
	}

	up.IsUsed = true
	return s.DB.Save(&up).Error
}

// List: คืนรายการโปรที่ผู้ใช้คนนี้ "เก็บไว้" พร้อมข้อมูลโปร (Preload("Promotion"))
// - เรียงใหม่สุดก่อน (id DESC)
// - Preload จะไม่ดึง Promotion ที่ถูก soft-delete อยู่แล้ว (ถ้า model ใช้ gorm.Model)
func (s *UserPromotionService) List(userID uint) ([]entity.UserPromotion, error) {
	var rows []entity.UserPromotion
	err := s.DB.
		Preload("Promotion").
		Where("user_id = ?", userID).
		Order("id DESC").
		Find(&rows).Error
	return rows, err
}
