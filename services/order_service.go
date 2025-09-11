package services

import (
	"backend/entity"
	"backend/repository"
	"errors"
	"time"

	"gorm.io/gorm"
)

// ---------- Config Struct ----------
type StatusIDs struct {
	Pending    uint
	Preparing  uint
	Delivering uint
	Completed  uint
	Cancelled  uint
}

type OrderService struct {
	DB       *gorm.DB
	Repo     *repository.OrderRepository
	CartRepo *repository.CartRepository
	RestRepo *repository.RestaurantRepository
	Status   StatusIDs
}

func NewOrderService(
	db *gorm.DB,
	repo *repository.OrderRepository,
	cartRepo *repository.CartRepository,
	restRepo *repository.RestaurantRepository,
) *OrderService {
	s := &OrderService{DB: db, Repo: repo, CartRepo: cartRepo, RestRepo: restRepo}

	// preload ค่า status id
	if id, err := repo.GetStatusIDByName("Pending"); err == nil {
		s.Status.Pending = id
	}
	if id, err := repo.GetStatusIDByName("Preparing"); err == nil {
		s.Status.Preparing = id
	}
	if id, err := repo.GetStatusIDByName("Delivering"); err == nil {
		s.Status.Delivering = id
	}
	if id, err := repo.GetStatusIDByName("Completed"); err == nil {
		s.Status.Completed = id
	}
	if id, err := repo.GetStatusIDByName("Cancelled"); err == nil {
		s.Status.Cancelled = id
	}
	return s
}

// ---------- DTOs ----------
type OrderItemIn struct {
	MenuID uint `json:"menuId"`
	Qty    int  `json:"qty"`
}

type CreateOrderReq struct {
	RestaurantID uint          `json:"restaurantId"`
	Items        []OrderItemIn `json:"items"`
	Discount     int64         `json:"discount,omitempty"`
	Address      string        `json:"address" binding:"required"`
}
type CreateOrderRes struct {
	ID    uint  `json:"id"`
	Total int64 `json:"total"`
}

type CheckoutFromCartReq struct {
	Address       string `json:"address" binding:"required"`
	PaymentMethod string `json:"paymentMethod" binding:"omitempty,oneof='PromptPay' 'Cash on Delivery'"`
	Discount      int64  `json:"discount,omitempty"`
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

// สำหรับ summary การจ่ายเงินล่าสุด
type PaymentSummary struct {
	MethodID   uint       `json:"methodId"`
	MethodName string     `json:"methodName"`
	StatusID   uint       `json:"statusId"`
	StatusName string     `json:"statusName"`
	PaidAt     *time.Time `json:"paidAt,omitempty"`
}

// ---------- Orders (CRUD หลัก) ----------

// POST /orders
func (s *OrderService) Create(userID uint, req *CreateOrderReq) (*CreateOrderRes, error) {
	if len(req.Items) == 0 {
		return nil, errors.New("items is required")
	}

	ok, err := s.Repo.RestaurantExists(req.RestaurantID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("restaurant not found")
	}

	// validate menus belong to restaurant
	menuIDs := make([]uint, 0, len(req.Items))
	for _, it := range req.Items {
		menuIDs = append(menuIDs, it.MenuID)
	}
	ok, err = s.Repo.ValidateMenusBelongToRestaurant(menuIDs, req.RestaurantID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("menu not in this restaurant")
	}

	// คำนวณ subtotal + prepare order items
	var subtotal int64
	rows := make([]struct {
		menuID    uint
		qty       int
		unitPrice int64
	}, 0, len(req.Items))
	for _, it := range req.Items {
		m, err := s.Repo.GetMenuBasics(it.MenuID)
		if err != nil {
			return nil, errors.New("menu not found")
		}
		unit := m.Price
		subtotal += unit * int64(it.Qty)
		rows = append(rows, struct {
			menuID    uint
			qty       int
			unitPrice int64
		}{m.ID, it.Qty, unit})
	}

	// discount logic
	discount := req.Discount
	if discount < 0 {
		discount = 0
	}
	if discount > subtotal {
		discount = subtotal
	}

	deliveryFee := int64(0)
	total := subtotal - discount + deliveryFee

	// Transaction
	var out CreateOrderRes
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		order := entity.Order{
			Subtotal:      subtotal,
			Discount:      discount,
			DeliveryFee:   deliveryFee,
			Total:         total,
			UserID:        userID,
			RestaurantID:  req.RestaurantID,
			OrderStatusID: s.Status.Pending,
			Address:       req.Address,
		}
		if err := s.Repo.CreateOrder(tx, &order); err != nil {
			return err
		}

		for _, r := range rows {
			oi := entity.OrderItem{
				Qty: r.qty, UnitPrice: r.unitPrice, Total: r.unitPrice * int64(r.qty),
				OrderID: order.ID, MenuID: r.menuID,
			}
			if err := s.Repo.CreateOrderItem(tx, &oi); err != nil {
				return err
			}
		}

		out = CreateOrderRes{ID: order.ID, Total: order.Total}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GET /orders (ลูกค้า)
func (s *OrderService) ListForUser(userID uint, limit int) ([]repository.OrderSummary, error) {
	return s.Repo.ListOrdersForUser(userID, limit)
}

// GET /orders/:id (ลูกค้า)
func (s *OrderService) DetailForUser(userID, orderID uint) (*OrderDetail, error) {
	o, err := s.Repo.GetOrderForUser(userID, orderID)
	if err != nil {
		return nil, err
	}
	items, err := s.Repo.GetOrderItems(o.ID)
	if err != nil {
		return nil, err
	}
	return &OrderDetail{
		ID: o.ID, Subtotal: o.Subtotal, Discount: o.Discount, DeliveryFee: o.DeliveryFee,
		Total: o.Total, OrderStatusID: o.OrderStatusID, RestaurantID: o.RestaurantID, Items: items,
	}, nil
}

// ---------- Payment ----------

// GET /orders/:id/payment-summary
func (s *OrderService) PaymentSummaryForOrder(userID, orderID uint) (*PaymentSummary, error) {
	// check สิทธิ์ก่อน
	if _, err := s.Repo.GetOrderForUser(userID, orderID); err != nil {
		return nil, err
	}

	var p entity.Payment
	if err := s.DB.
		Where("order_id = ?", orderID).
		Order("id DESC").
		Preload("PaymentMethod").
		Preload("PaymentStatus").
		First(&p).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &PaymentSummary{
		MethodID:   p.PaymentMethodID,
		MethodName: p.PaymentMethod.MethodName,
		StatusID:   p.PaymentStatusID,
		StatusName: p.PaymentStatus.StatusName,
		PaidAt:     p.PaidAt,
	}, nil
}

// ---------- Orders from Cart ----------

// POST /checkout
func (s *OrderService) CreateFromCart(userID uint, in *CheckoutFromCartReq) (*CreateOrderRes, error) {
	cart, err := s.CartRepo.GetCartWithItems(userID)
	if err != nil {
		return nil, err
	}
	if cart.RestaurantID == 0 {
		return nil, errors.New("cart has no restaurant")
	}
	if len(cart.Items) == 0 {
		return nil, errors.New("cart is empty")
	}

	// subtotal
	var subtotal int64
	for _, it := range cart.Items {
		subtotal += it.Total
	}

	// discount
	discount := in.Discount
	if discount < 0 {
		discount = 0
	}
	if discount > subtotal {
		discount = subtotal
	}

	delivery := int64(0)
	total := subtotal - discount + delivery

	// Transaction
	var out CreateOrderRes
	err = s.DB.Transaction(func(tx *gorm.DB) error {
		order := entity.Order{
			UserID:        userID,
			RestaurantID:  cart.RestaurantID,
			OrderStatusID: s.Status.Pending,
			Subtotal:      subtotal,
			Discount:      discount,
			DeliveryFee:   delivery,
			Total:         total,
			Address:       in.Address,
		}
		if err := s.Repo.CreateOrder(tx, &order); err != nil {
			return err
		}

		// copy items
		for _, it := range cart.Items {
			oi := entity.OrderItem{
				OrderID:   order.ID,
				MenuID:    it.MenuID,
				Qty:       it.Qty,
				UnitPrice: it.UnitPrice,
				Total:     it.Total,
			}
			if err := s.Repo.CreateOrderItem(tx, &oi); err != nil {
				return err
			}
		}

		// optional payment record
		if in.PaymentMethod != "" {
			pmID, err := s.Repo.GetPaymentMethodIDFromKey(in.PaymentMethod)
			if err != nil {
				return err
			}
			if pmID != 0 {
				const payPending uint = 1
				p := entity.Payment{
					Amount:          total,
					OrderID:         order.ID,
					PaymentMethodID: pmID,
					PaymentStatusID: payPending,
				}
				if err := s.Repo.CreatePayment(tx, &p); err != nil {
					return err
				}
			}
		}

		// clear cart
		if err := s.CartRepo.ClearCart(tx, userID); err != nil {
			return err
		}

		out = CreateOrderRes{ID: order.ID, Total: order.Total}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ---------- Owner (ร้าน) ----------

func (s *OrderService) RepoOwnerCheck(restID, userID uint) (bool, error) {
	return s.RestRepo.IsOwnedBy(restID, userID)
}

type OwnerOrderListOut struct {
	Items []repository.OwnerOrderSummary `json:"items"`
	Total int64                          `json:"total"`
	Page  int                            `json:"page"`
	Limit int                            `json:"limit"`
}

// GET /owner/restaurants/:id/orders
func (s *OrderService) ListForRestaurant(userID, restID uint, statusID *uint, page, limit int) (*OwnerOrderListOut, error) {
	ok, err := s.RepoOwnerCheck(restID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("forbidden")
	}

	items, total, err := s.Repo.ListOrdersForRestaurant(restID, statusID, page, limit)
	if err != nil {
		return nil, err
	}
	return &OwnerOrderListOut{Items: items, Total: total, Page: page, Limit: limit}, nil
}

type OwnerOrderDetail struct {
	Order entity.Order       `json:"order"`
	Items []entity.OrderItem `json:"items"`
}

// GET /owner/restaurants/:id/orders/:oid
func (s *OrderService) DetailForRestaurant(userID, restID, orderID uint) (*OwnerOrderDetail, error) {
	ok, err := s.RepoOwnerCheck(restID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("forbidden")
	}

	o, err := s.Repo.GetOrderForRestaurant(restID, orderID)
	if err != nil {
		return nil, err
	}

	items, err := s.Repo.GetOrderItems(o.ID)
	if err != nil {
		return nil, err
	}

	return &OwnerOrderDetail{Order: *o, Items: items}, nil
}
