package entity

import (
	"time"

	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	Amount          int64      `json:"amount"` // หน่วยสตางค์
	PaidAt          *time.Time `json:"paidAt,omitempty"`
	SlipContentType string     `gorm:"type:varchar(64)" json:"slipContentType,omitempty"`
	SlipBase64      string     `gorm:"type:longtext" json:"slipBase64,omitempty"` //เก็บ base64
	TransRef        string     `gorm:"size:100;uniqueIndex" json:"transRef,omitempty"`

	PaymentMethodID uint          `json:"paymentMethodId"`
	PaymentMethod   PaymentMethod `json:"-"`

	OrderID uint  `gorm:"uniqueIndex" json:"orderId"`
	Order   Order `json:"-"`

	PaymentStatusID uint          `json:"paymentStatusId"`
	PaymentStatus   PaymentStatus `json:"-"`
}
