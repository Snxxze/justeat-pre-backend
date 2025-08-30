package entity

import (
	"time"
	"gorm.io/gorm"
)

type Payment struct {
	gorm.Model
	Amount  int64      `json:"amount"`
	SlipURL string     `json:"slipUrl"`
	PaidAt  *time.Time `json:"paidAt,omitempty"`

	PaymentMethodID uint          `json:"paymentMethodId"`
	PaymentMethod   PaymentMethod `json:"-"` // preload เฉพาะตอนต้องการชื่อ method

	OrderID uint  `json:"orderId"`
	Order   Order `json:"-"` // preload เฉพาะ endpoint /orders/:id

	PaymentStatusID uint          `json:"paymentStatusId"`
	PaymentStatus   PaymentStatus `json:"-"` // preload เฉพาะตอน detail
}

