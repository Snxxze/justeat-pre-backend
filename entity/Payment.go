package entity

import (
	"time"
	"gorm.io/gorm"
)

type Payment struct {
    gorm.Model
    Amount  int64      `json:"amount"`
    SlipURL string     `json:"slipUrl"`         // ของเดิม
    SlipBase64 string  `gorm:"type:longtext" json:"slipBase64,omitempty"` // ของใหม่
	SlipContentType string `gorm:"type:varchar(64)" json:"slipContentType,omitempty"`

    PaidAt  *time.Time `json:"paidAt,omitempty"`

    PaymentMethodID uint          `json:"paymentMethodId"`
    PaymentMethod   PaymentMethod `json:"-"`

    OrderID uint  `json:"orderId"`
    Order   Order `json:"-"`

    PaymentStatusID uint          `json:"paymentStatusId"`
    PaymentStatus   PaymentStatus `json:"-"`
}
