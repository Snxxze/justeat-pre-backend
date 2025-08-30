package entity

import (
	"time"
	"gorm.io/gorm"
)

// ใบสมัครเปิดร้าน โดยยัง "ไม่" สร้างร้านจริงจนกว่าจะอนุมัติ
type RestaurantApplication struct {
	gorm.Model
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
	Picture     string `json:"picture"`

	RestaurantCategoryID uint `json:"restaurantCategoryId"`
	OwnerUserID          uint `json:"ownerUserId"` // คนยื่น (เจ้าของในอนาคต)

	// pending / approved / rejected
	Status string `gorm:"not null;default:pending" json:"status"`

	AdminID      *uint      `json:"adminId,omitempty"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	RejectReason *string    `json:"rejectReason,omitempty"`
}
