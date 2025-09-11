package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

type PaymentRepository struct {
	DB *gorm.DB
}

func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{DB: db}
}

// ดึง Payment จาก OrderID
func (r *PaymentRepository) GetByOrderID(orderID uint) (*entity.Payment, error) {
	var p entity.Payment
	err := r.DB.Where("order_id = ?", orderID).First(&p).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// อัปเดตสถานะ Payment (+ optional PaidAt)
func (r *PaymentRepository) UpdateStatus(tx *gorm.DB, paymentID, statusID uint, paidAt *time.Time) error {
	updates := map[string]any{
		"payment_status_id": statusID,
	}
	if paidAt != nil {
		updates["paid_at"] = paidAt
	}
	return tx.Model(&entity.Payment{}).Where("id = ?", paymentID).Updates(updates).Error
}

// คืนค่า PaymentMethodID ตามชื่อ
func (r *PaymentRepository) GetMethodIDByName(name string) (uint, error) {
    var id uint
    err := r.DB.Model(&entity.PaymentMethod{}).Select("id").Where("method_name = ?", name).Scan(&id).Error
    return id, err
}

// คืนค่า PaymentStatusID ตามชื่อ
func (r *PaymentRepository) GetStatusIDByName(name string) (uint, error) {
    var id uint
    err := r.DB.Model(&entity.PaymentStatus{}).Select("id").Where("status_name = ?", name).Scan(&id).Error
    return id, err
}
