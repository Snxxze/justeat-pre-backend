package services

import (
	"backend/entity"
	"backend/repository"
	"errors"

	"gorm.io/gorm"
)

type CartService struct {
	DB       *gorm.DB
	CartRepo *repository.CartRepository
	OrderRepo *repository.OrderRepository // reuse validate menu/option + ราคา
}

func NewCartService(db *gorm.DB, cr *repository.CartRepository, or *repository.OrderRepository) *CartService {
	return &CartService{DB: db, CartRepo: cr, OrderRepo: or}
}

type AddToCartIn struct {
	RestaurantID uint   `json:"restaurantId" binding:"required"`
	MenuID       uint   `json:"menuId" binding:"required"`
	Qty          int    `json:"qty" binding:"min=1"`
	Note         string `json:"note"`
	Selections   []struct {
		OptionValueID uint `json:"optionValueId" binding:"required"`
	} `json:"selections"`
}

func (s *CartService) Get(userID uint) (*entity.Cart, int64, error) {
	c, err := s.CartRepo.GetCartWithItems(userID)
	if err != nil { return nil, 0, err }
	var subtotal int64
	for _, it := range c.Items { subtotal += it.Total }
	return c, subtotal, nil
}

func (s *CartService) Add(userID uint, in *AddToCartIn) error {
	if in.Qty <= 0 { in.Qty = 1 }

	c, err := s.CartRepo.GetOrCreateCart(userID, in.RestaurantID)
	if err != nil { return err }

	// ถ้าคาร์ทเคยล็อกร้านอื่นไว้ และยังไม่ถูกรีเซ็ต -> ไม่ให้ข้ามร้าน
	if c.RestaurantID != 0 && c.RestaurantID != in.RestaurantID {
		return errors.New("cart has another restaurant")
	}
	// ถ้าคาร์ทยังไม่ล็อกร้าน (เช่นเพิ่งถูกล้าง) ให้ตั้งร้านใหม่
	if c.RestaurantID == 0 {
		c.RestaurantID = in.RestaurantID
		if err := s.CartRepo.DB.Save(c).Error; err != nil { return err }
	}

	// ตรวจเมนู + คำนวณราคา
	m, err := s.OrderRepo.GetMenuBasics(in.MenuID)
	if err != nil { return err }

	// ✅ ยืนยันเมนูอยู่ในร้านเดียวกับ cart
	if m.RestaurantID != in.RestaurantID {
		return errors.New("menu not in this restaurant")
	}

	valIDs := make([]uint, 0, len(in.Selections))
	for _, s := range in.Selections { valIDs = append(valIDs, s.OptionValueID) }
	if len(valIDs) > 0 {
		cnt, err := s.OrderRepo.CountOptionValuesBelongToMenu(m.ID, valIDs)
		if err != nil { return err }
		if cnt != int64(len(valIDs)) {
			return errors.New("invalid option values")
		}
	}
	vals, err := s.OrderRepo.GetOptionValuesByIDs(valIDs)
	if err != nil { return err }

	unit := m.Price
	selRows := make([]entity.CartItemSelection, 0, len(vals))
	for _, v := range vals {
		unit += v.PriceAdjustment
		selRows = append(selRows, entity.CartItemSelection{
			OptionID: v.OptionID, OptionValueID: v.ID, PriceDelta: v.PriceAdjustment,
		})
	}

	line := &entity.CartItem{
		MenuID: m.ID, Qty: in.Qty, UnitPrice: unit, Total: unit * int64(in.Qty), Note: in.Note,
		Selections: selRows,
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		return s.CartRepo.UpsertItem(tx, c.ID, line)
	})
}


func (s *CartService) UpdateQty(userID, itemID uint, qty int) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return s.CartRepo.UpdateQty(tx, userID, itemID, qty)
	})
}

func (s *CartService) RemoveItem(userID, itemID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return s.CartRepo.RemoveItem(tx, userID, itemID)
	})
}

func (s *CartService) Clear(userID uint) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		return s.CartRepo.ClearCart(tx, userID)
	})
}
