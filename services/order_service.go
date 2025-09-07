package services

import (
	"errors"
	"backend/entity"
	"backend/repository"

	"gorm.io/gorm"
)

var (
	ErrEmptyItems           = errors.New("empty items")
	ErrRestaurantNotFound   = errors.New("restaurant not found")
	ErrMenuNotFound         = errors.New("menu not found")
	ErrMenuNotInRestaurant  = errors.New("menu not in this restaurant")
	ErrInvalidOptionValue   = errors.New("invalid option values for menu")
)

type OrderService struct {
	DB   *gorm.DB
	Repo *repository.OrderRepository
}

func NewOrderService(db *gorm.DB, repo *repository.OrderRepository) *OrderService {
	return &OrderService{DB: db, Repo: repo}
}

// ----- DTOs from Controller -----
type OrderItemSelectionIn struct {
	OptionID      uint `json:"optionId"`
	OptionValueID uint `json:"optionValueId"`
}
type OrderItemIn struct {
	MenuID     uint                   `json:"menuId"`
	Qty        int                    `json:"qty"`
	Selections []OrderItemSelectionIn `json:"selections"`
}
type CreateOrderReq struct {
	RestaurantID uint          `json:"restaurantId"`
	Items        []OrderItemIn `json:"items"`
}

type CreateOrderRes struct {
	ID    uint  `json:"id"`
	Total int64 `json:"total"`
}

// ----- Create -----
func (s *OrderService) Create(userID uint, req *CreateOrderReq) (*CreateOrderRes, error) {
	if len(req.Items) == 0 {
		return nil, ErrEmptyItems
	}
	// ร้านต้องมีจริง
	ok, err := s.Repo.RestaurantExists(req.RestaurantID)
	if err != nil { return nil, err }
	if !ok { return nil, ErrRestaurantNotFound }

	// เมนูทั้งหมดต้องอยู่ร้านนี้
	menuIDs := make([]uint, 0, len(req.Items))
	for _, it := range req.Items { menuIDs = append(menuIDs, it.MenuID) }
	ok, err = s.Repo.ValidateMenusBelongToRestaurant(menuIDs, req.RestaurantID)
	if err != nil { return nil, err }
	if !ok { return nil, ErrMenuNotInRestaurant }

	// คำนวณราคา & เตรียม payload
	type calc struct {
		menuID    uint
		qty       int
		unitPrice int64
		sels      []OrderItemSelectionIn
		vals      []entity.OptionValue
	}
	rows := make([]calc, 0, len(req.Items))
	var subtotal int64

	for _, it := range req.Items {
		m, err := s.Repo.GetMenuBasics(it.MenuID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) { return nil, ErrMenuNotFound }
			return nil, err
		}

		// ตรวจสอบ option values เป็นของเมนูนี้จริง
		var valueIDs []uint
		for _, s := range it.Selections { valueIDs = append(valueIDs, s.OptionValueID) }
		if len(valueIDs) > 0 {
			cnt, err := s.Repo.CountOptionValuesBelongToMenu(m.ID, valueIDs)
			if err != nil { return nil, err }
			if cnt != int64(len(valueIDs)) { return nil, ErrInvalidOptionValue }
		}

		// ดึงรายละเอียด option values เพื่อรู้ price delta
		vals, err := s.Repo.GetOptionValuesByIDs(valueIDs)
		if err != nil { return nil, err }

		unit := m.Price
		for _, v := range vals { unit += v.PriceAdjustment }
		subtotal += unit * int64(it.Qty)

		rows = append(rows, calc{
			menuID: m.ID, qty: it.Qty, unitPrice: unit, sels: it.Selections, vals: vals,
		})
	}

	discount := int64(0)     // MVP
	deliveryFee := int64(20) // MVP: flat 20
	total := subtotal - discount + deliveryFee
	const pendingStatusID uint = 1 // TODO: ควร lookup จาก table order_statuses

	var out CreateOrderRes
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		order := entity.Order{
			Subtotal: subtotal, Discount: discount, DeliveryFee: deliveryFee, Total: total,
			UserID: userID, RestaurantID: req.RestaurantID, OrderStatusID: pendingStatusID,
		}
		if err := s.Repo.CreateOrder(tx, &order); err != nil { return err }

		for _, r := range rows {
			oi := entity.OrderItem{
				Qty: r.qty, UnitPrice: r.unitPrice, Total: r.unitPrice * int64(r.qty),
				OrderID: order.ID, MenuID: r.menuID,
			}
			if err := s.Repo.CreateOrderItem(tx, &oi); err != nil { return err }

			// เก็บ selections (บันทึก PriceDelta ไว้เพื่ออ้างอิง)
			ins := make([]entity.OrderItemSelection, 0, len(r.sels))
			// map ไอดี -> delta
			delta := map[uint]int64{}
			for _, v := range r.vals { delta[v.ID] = v.PriceAdjustment }
			for _, sIn := range r.sels {
				ins = append(ins, entity.OrderItemSelection{
					OrderItemID: oi.ID,
					OptionID: sIn.OptionID,
					OptionValueID: sIn.OptionValueID,
					PriceDelta: delta[sIn.OptionValueID],
				})
			}
			if err := s.Repo.CreateOrderItemSelections(tx, ins); err != nil { return err }
		}

		out = CreateOrderRes{ ID: order.ID, Total: order.Total }
		return nil
	})

	if err != nil { return nil, err }
	return &out, nil
}

// ----- List & Detail -----
func (s *OrderService) ListForUser(userID uint, limit int) ([]repository.OrderSummary, error) {
	return s.Repo.ListOrdersForUser(userID, limit)
}

type OrderDetail struct {
	ID            uint               `json:"id"`
	Subtotal      int64              `json:"subtotal"`
	Discount      int64              `json:"discount"`
	DeliveryFee   int64              `json:"deliveryFee"`
	Total         int64              `json:"total"`
	OrderStatusID uint               `json:"orderStatusId"`
	RestaurantID  uint               `json:"restaurantId"`
	Items         []entity.OrderItem `json:"items"`
}

func (s *OrderService) DetailForUser(userID, orderID uint) (*OrderDetail, error) {
	o, err := s.Repo.GetOrderForUser(userID, orderID)
	if err != nil { return nil, err }
	items, err := s.Repo.GetOrderItems(o.ID)
	if err != nil { return nil, err }
	return &OrderDetail{
		ID: o.ID, Subtotal: o.Subtotal, Discount: o.Discount, DeliveryFee: o.DeliveryFee,
		Total: o.Total, OrderStatusID: o.OrderStatusID, RestaurantID: o.RestaurantID, Items: items,
	}, nil
}
