package controllers

import (
	"errors"
	"net/http"

	"backend/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CartController struct {
	DB *gorm.DB
}

func NewCartController(db *gorm.DB) *CartController { return &CartController{DB: db} }

// ========================
// Helpers เฉพาะงานฐานข้อมูล
// ========================

// ดึงตะกร้าพร้อมรายการของผู้ใช้; ถ้าไม่มีใน DB จะคืน struct เปล่าสำหรับแสดงผลได้
func (h *CartController) getCartWithItems(userID uint) (*entity.Cart, error) {
	var cart entity.Cart
	err := h.DB.Where("user_id = ?", userID).
		Preload("Items").
		Preload("Items.Menu").
		First(&cart).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		// ยังไม่มี cart ใน DB → คืนค่าว่าง (แต่อิง userID)
		return &entity.Cart{UserID: userID}, nil
	}
	return &cart, err
}

// สร้างหรือดึง Cart ของผู้ใช้ (ถ้ายังไม่มีก็สร้างพร้อมตั้งร้าน)
func (h *CartController) getOrCreateCart(userID, restaurantID uint) (*entity.Cart, error) {
	var cart entity.Cart
	err := h.DB.Where("user_id = ?", userID).First(&cart).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		cart = entity.Cart{UserID: userID, RestaurantID: restaurantID}
		if err := h.DB.Create(&cart).Error; err != nil {
			return nil, err
		}
		return &cart, nil
	}
	return &cart, err
}

// ถ้าตะกร้าไม่มี item แล้ว → reset restaurant_id = 0 เพื่อพร้อมรับร้านใหม่
func (h *CartController) resetRestaurantIfEmpty(tx *gorm.DB, cartID uint) error {
	var remainingItemCount int64
	if err := tx.Model(&entity.CartItem{}).Where("cart_id = ?", cartID).Count(&remainingItemCount).Error; err != nil {
		return err
	}
	if remainingItemCount == 0 {
		return tx.Model(&entity.Cart{}).Where("id = ?", cartID).Update("restaurant_id", 0).Error
	}
	return nil
}

// ========================
// Handlers
// ========================

// GET /cart
func (h *CartController) Get(c *gin.Context) {
	// ดึง userId จาก context แบบตรง ๆ
	value, exists := c.Get("userId")
	if !exists || value == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID := value.(uint)
	if currentUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	cart, err := h.getCartWithItems(currentUserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var subtotal int64
	for _, item := range cart.Items {
		subtotal += item.Total
	}

	c.JSON(http.StatusOK, gin.H{"cart": cart, "subtotal": subtotal})
}

// POST /cart/items
func (h *CartController) Add(c *gin.Context) {
	// --- Extract userId ---
	value, exists := c.Get("userId")
	if !exists || value == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID := value.(uint)
	if currentUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	// --- Bind JSON ---
	type AddItemRequest struct {
		RestaurantID uint   `json:"restaurantId" binding:"required"`
		MenuID       uint   `json:"menuId" binding:"required"`
		Quantity     int    `json:"qty" binding:"min=1"`
		Note         string `json:"note"`
	}
	var requestBody AddItemRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if requestBody.Quantity <= 0 {
		requestBody.Quantity = 1
	}

	// --- หา/สร้าง cart ---
	var cart entity.Cart
	if err := h.DB.Where("user_id = ?", currentUserID).First(&cart).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cart = entity.Cart{UserID: currentUserID, RestaurantID: requestBody.RestaurantID}
			if err := h.DB.Create(&cart).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	// --- ถ้ามีร้านอื่นอยู่ใน cart → ล้างก่อน ---
	if cart.RestaurantID != 0 && cart.RestaurantID != requestBody.RestaurantID {
		if err := h.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("cart_id = ?", cart.ID).Delete(&entity.CartItem{}).Error; err != nil {
				return err
			}
			return tx.Model(&entity.Cart{}).Where("id = ?", cart.ID).
				Update("restaurant_id", requestBody.RestaurantID).Error
		}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		cart.RestaurantID = requestBody.RestaurantID
	}

	// --- โหลดเมนู ---
	var menu entity.Menu
	if err := h.DB.Select("id, price, restaurant_id").First(&menu, requestBody.MenuID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "menu not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if menu.RestaurantID != requestBody.RestaurantID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "menu not in this restaurant"})
		return
	}

	// --- Upsert Item ---
	if err := h.DB.Transaction(func(tx *gorm.DB) error {
		var existingItem entity.CartItem
		findErr := tx.Where("cart_id = ? AND menu_id = ? AND note = ?", cart.ID, requestBody.MenuID, requestBody.Note).
			First(&existingItem).Error

		if findErr == nil {
			existingItem.Qty += requestBody.Quantity
			existingItem.Total = int64(existingItem.Qty) * menu.Price
			return tx.Save(&existingItem).Error
		}
		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return findErr
		}

		newItem := entity.CartItem{
			CartID:    cart.ID,
			MenuID:    requestBody.MenuID,
			Qty:       requestBody.Quantity,
			UnitPrice: menu.Price,
			Total:     menu.Price * int64(requestBody.Quantity),
			Note:      requestBody.Note,
		}
		return tx.Create(&newItem).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true})
}

// PATCH /cart/items/qty
func (h *CartController) UpdateQty(c *gin.Context) {
	value, exists := c.Get("userId")
	if !exists || value == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID := value.(uint)
	if currentUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	type UpdateQtyRequest struct {
		ItemID   uint `json:"itemId" binding:"required"`
		Quantity int  `json:"qty" binding:"required"`
	}
	var requestBody UpdateQtyRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// โหลด item โดย join carts เพื่อตรวจ ownership ของผู้ใช้
		var cartItem entity.CartItem
		if e := tx.
			Joins("JOIN carts ON carts.id = cart_items.cart_id").
			Where("cart_items.id = ? AND carts.user_id = ?", requestBody.ItemID, currentUserID).
			Select("cart_items.*").
			First(&cartItem).Error; e != nil {
			return e
		}

		// ถ้า qty <= 0 → ถือว่าเป็นการลบ
		if requestBody.Quantity <= 0 {
			if e := tx.Delete(&entity.CartItem{}, cartItem.ID).Error; e != nil {
				return e
			}
			return h.resetRestaurantIfEmpty(tx, cartItem.CartID)
		}

		// อัปเดต qty + total (unit_price เป็น snapshot ใน cart item)
		return tx.Model(&entity.CartItem{}).
			Where("id = ?", cartItem.ID).
			Updates(map[string]any{
				"qty":   requestBody.Quantity,
				"total": cartItem.UnitPrice * int64(requestBody.Quantity),
			}).Error
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /cart/items
func (h *CartController) RemoveItem(c *gin.Context) {
	value, exists := c.Get("userId")
	if !exists || value == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID := value.(uint)
	if currentUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	type RemoveItemRequest struct {
		ItemID uint `json:"itemId" binding:"required"`
	}
	var requestBody RemoveItemRequest
	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// ตรวจว่ารายการเป็นของ cart ของ user จริง
		var cartItem entity.CartItem
		if e := tx.
			Joins("JOIN carts ON carts.id = cart_items.cart_id").
			Where("cart_items.id = ? AND carts.user_id = ?", requestBody.ItemID, currentUserID).
			Select("cart_items.*").
			First(&cartItem).Error; e != nil {
			return e
		}

		// ลบรายการ
		if e := tx.Delete(&entity.CartItem{}, cartItem.ID).Error; e != nil {
			return e
		}

		// ถ้าตะกร้ากลายเป็นว่าง → reset restaurant_id = 0
		return h.resetRestaurantIfEmpty(tx, cartItem.CartID)
	})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /cart
func (h *CartController) Clear(c *gin.Context) {
	value, exists := c.Get("userId")
	if !exists || value == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	currentUserID := value.(uint)
	if currentUserID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	err := h.DB.Transaction(func(tx *gorm.DB) error {
		// หา cart ของ user (idempotent: ไม่มี cart ก็ถือว่าเคลียร์แล้ว)
		var cart entity.Cart
		if e := tx.Where("user_id = ?", currentUserID).First(&cart).Error; e != nil {
			if errors.Is(e, gorm.ErrRecordNotFound) {
				return nil
			}
			return e
		}

		// ลบ items ทั้งหมดใน cart
		if e := tx.Where("cart_id = ?", cart.ID).Delete(&entity.CartItem{}).Error; e != nil {
			return e
		}

		// รีเซ็ตให้พร้อมรับร้านใหม่
		return tx.Model(&entity.Cart{}).Where("id = ?", cart.ID).Update("restaurant_id", 0).Error
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
