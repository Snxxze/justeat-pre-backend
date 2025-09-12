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
	Phone 			string `json:"phone"`
	Description string `json:"description"`
	Picture     string `json:"pictureBase64,omitempty" gorm:"column:picture_base64;type:longtext"`

	OpeningTime string `json:"openingTime"`
	ClosingTime string `json:"closingTime"`

	RestaurantCategoryID uint `json:"restaurantCategoryId"`
	RestaurantCategory   RestaurantCategory `json:"restaurantCategory" gorm:"foreignKey:RestaurantCategoryID"`

	PromptPay string `json:"promptPay" gorm:"column:prompt_pay;type:varchar(32)"`

	OwnerUserID          uint `json:"ownerUserId"` // คนยื่น (เจ้าของในอนาคต)
	OwnerUser   User   `json:"ownerUser"` // preload เอามาโชว์

	// pending / approved / rejected
	Status string `gorm:"not null;default:pending" json:"status"`

	AdminID      *uint      `json:"adminId,omitempty"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	RejectReason *string    `json:"rejectReason,omitempty"`
}
