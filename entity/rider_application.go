// entity/rider_application.go
package entity

import (
	"time"
	"gorm.io/gorm"
)

// ใบสมัคร Rider: ยัง "ไม่" สร้าง Rider จริงจนกว่าอนุมัติ
type RiderApplication struct {
	gorm.Model
	VehiclePlate string `json:"vehiclePlate"`
	License      string `json:"license"`
	DriveCar     string `json:"driveCarPicture" gorm:"column:drive_car_picture;type:longtext"` // base64

	// ใครยื่น
	UserID uint `json:"userId" gorm:"index"`
	User   User `json:"user" gorm:"foreignKey:UserID;references:ID"`

	// สถานะ: pending / approved / rejected
	Status string `gorm:"not null;default:pending" json:"status"`

	AdminID      *uint      `json:"adminId,omitempty"`
	ReviewedAt   *time.Time `json:"reviewedAt,omitempty"`
	RejectReason *string    `json:"rejectReason,omitempty"`
}