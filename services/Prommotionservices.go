package services

import (
	"backend/configs"
	"backend/entity"
	"log"
)

type PromotionService struct{
    
}

func NewPromotionService() *PromotionService {
    return &PromotionService{}
}

func (s *PromotionService) CreatePromotion(promo *entity.Promotion, adminID uint) error {
    promo.AdminID = adminID
    log.Println("Attempting to create promotion:", promo)
    err := configs.DB().Create(promo).Error
    if err != nil {
        log.Println("Error creating promotion:", err)
    }
    return err
}

func (s *PromotionService) GetAllPromotions() ([]entity.Promotion, error) {
    var promotions []entity.Promotion
    // ไม่จำเป็นต้อง preload "Picture" อีกต่อไป เพราะไม่มีการอัปโหลดรูปภาพ
    err := configs.DB().Preload("PromoType").Preload("Admin.User").Find(&promotions).Error
    return promotions, err
}

func (s *PromotionService) UpdatePromotion(id uint, promo *entity.Promotion) error {
    var existingPromo entity.Promotion
    if err := configs.DB().First(&existingPromo, id).Error; err != nil {
        return err
    }
    // ลบส่วนที่เกี่ยวข้องกับการอัปเดตรูปภาพออก
    return configs.DB().Model(&existingPromo).Updates(promo).Error
}

func (s *PromotionService) DeletePromotion(id uint) error {
    return configs.DB().Delete(&entity.Promotion{}, id).Error
}

// ลบฟังก์ชัน SaveUploadedFile ออกทั้งหมด เนื่องจากไม่มีการใช้งานแล้ว
// func SaveUploadedFile(file *multipart.FileHeader) (string, error) {
// ...
// }