package repository

import (
	"backend/entity"
	"errors"

	"gorm.io/gorm"
)

type CartRepository struct{ DB *gorm.DB }

func NewCartRepository(db *gorm.DB) *CartRepository { return &CartRepository{DB: db} }

// คืน Cart เดิมของ user (ถ้าไม่มีก็คืน Cart ว่าง ๆ โดยไม่ error เพื่อให้ FE แสดงได้)
func (r *CartRepository) GetCartWithItems(userID uint) (*entity.Cart, error) {
	var c entity.Cart
	err := r.DB.Where("user_id = ?", userID).
		Preload("Items").
		Preload("Items.Menu").
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &entity.Cart{UserID: userID}, nil
	}
	return &c, err
}

// สร้างหรืออ่าน Cart ของ user (และตั้ง RestaurantID ถ้าเพิ่งสร้าง)
func (r *CartRepository) GetOrCreateCart(userID, restaurantID uint) (*entity.Cart, error) {
	var c entity.Cart
	err := r.DB.Where("user_id = ?", userID).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c = entity.Cart{UserID: userID, RestaurantID: restaurantID}
		if err := r.DB.Create(&c).Error; err != nil {
			return nil, err
		}
		return &c, nil
	}
	return &c, err
}

// เพิ่มหรือรวม line: ตัวอย่าง merge แบบง่าย (เมนูเดียวกัน + note เดียวกัน)
// **หมายเหตุ** ถ้าต้องรวมตาม selections ด้วย ให้เก็บ hash selections ไว้คอลัมน์เพิ่ม แล้วใช้ในเงื่อนไข Where
func (r *CartRepository) UpsertItem(tx *gorm.DB, cartID uint, row *entity.CartItem) error {
	var exist entity.CartItem
	err := tx.Where("cart_id = ? AND menu_id = ? AND note = ?", cartID, row.MenuID, row.Note).
		First(&exist).Error
	if err == nil {
		exist.Qty += row.Qty
		exist.Total = int64(exist.Qty) * exist.UnitPrice
		return tx.Save(&exist).Error
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}

	row.CartID = cartID
	if err := tx.Create(row).Error; err != nil {
		return err
	}
	return nil
}

func (r *CartRepository) UpdateQty(tx *gorm.DB, userID, itemID uint, qty int) error {
	if qty <= 0 {
		return r.RemoveItem(tx, userID, itemID)
	}
	// ensure item เป็นของ cart ของ user
	return tx.Exec(`
		UPDATE cart_items
		   SET qty = ?, total = unit_price * ?
		 WHERE id = ?
		   AND cart_id IN (SELECT id FROM carts WHERE user_id = ?)
	`, qty, qty, itemID, userID).Error
}

func (r *CartRepository) RemoveItem(tx *gorm.DB, userID, itemID uint) error {
	// ลบรายการ
	if err := tx.
		Where("id = ? AND cart_id IN (SELECT id FROM carts WHERE user_id = ?)", itemID, userID).
		Delete(&entity.CartItem{}).Error; err != nil {
		return err
	}
	// ถ้าตะกร้าว่างแล้ว → รีเซ็ต restaurant_id = 0
	return tx.Exec(`
		UPDATE carts SET restaurant_id = 0
		 WHERE user_id = ?
		   AND NOT EXISTS (SELECT 1 FROM cart_items ci WHERE ci.cart_id = carts.id)
	`, userID).Error
}

func (r *CartRepository) ClearCart(tx *gorm.DB, userID uint) error {
	var c entity.Cart
	if err := tx.Where("user_id = ?", userID).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { return nil }
		return err
	}
	if err := tx.Where("cart_id = ?", c.ID).Delete(&entity.CartItem{}).Error; err != nil {
		return err
	}
	// รีเซ็ตร้านของตะกร้าให้เป็น 0 เพื่อพร้อมรับร้านใหม่
  if err := tx.Model(&entity.Cart{}).Where("id = ?", c.ID).Update("restaurant_id", 0).Error; err != nil {
    return err
  }
  return nil
}
