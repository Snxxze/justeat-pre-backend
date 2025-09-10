package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"backend/entity"
	"backend/pkg/resp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct{ 
	DB *gorm.DB 
}

func NewAdminController(db *gorm.DB) *AdminController { 
	return &AdminController{DB: db} 
}

// Dashboard: ตัวเลขรวม ๆ
func (ac *AdminController) Dashboard(c *gin.Context) {
	db := ac.DB

	// ตัวแปรผลลัพธ์
	var totalUsers int64
	var totalRestaurants int64
	var pendingApps int64
	var ordersToday int64

	// ผู้ใช้ทั้งหมด
	if err := db.Model(&entity.User{}).Count(&totalUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "count users failed"})
		return
	}

	// ร้านทั้งหมด
	if err := db.Model(&entity.Restaurant{}).Count(&totalRestaurants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "count restaurants failed"})
		return
	}

	// ใบสมัครร้านที่รออนุมัติ
	if err := db.Model(&entity.RestaurantApplication{}).
		Where("status = ?", "pending").
		Count(&pendingApps).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "count pending application failed"})
			return
		}

		// นับออเดอร์ของ วันนี้
		start := time.Now().Truncate(24 * time.Hour)
		if err := db.Model(&entity.Order{}).
			Where("created_at >= ?", start).
			Count(&ordersToday).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "count orders today failed"})
				return
			}

		// ตอบกลับ
		c.JSON(http.StatusOK, gin.H{
			"totalUser":						totalUsers,
			"totalRestaurants":			totalRestaurants,
			"pendingApplications":	pendingApps,
			"ordersToday":					ordersToday,
		})
}

// รายการร้าน (page/limit)
func (ac *AdminController) Restaurants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 { page = 1 }
	if limit <= 0 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	var total int64
	ac.DB.Model(&entity.Restaurant{}).Count(&total)

	type row struct {
		ID uint `json:"id"`
		Name string `json:"name"`
		RestaurantStatusID uint `json:"restaurantStatusId"`
		UserID uint `json:"ownerUserId"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var items []row
	if err := ac.DB.Model(&entity.Restaurant{}).
		Select("id, name, restaurant_status_id, user_id, created_at").
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items, "page": page, "limit": limit, "total": total})
}

// รายงานปัญหา
func (ac *AdminController) Reports(c *gin.Context) {
	type row struct {
		ID uint `json:"id"`
		Name string `json:"name"`
		IssueTypeID uint `json:"issueTypeId"`
		UserID uint `json:"userId"`
		DateAt *time.Time `json:"dateAt,omitempty"`
		CreatedAt time.Time `json:"createdAt"`
	}
	var items []row
	if err := ac.DB.Model(&entity.Report{}).
		Select("id, name, issue_type_id, user_id, date_at, created_at").
		Order("id DESC").Limit(100).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items})
}

// ไรเดอร์
func (ac *AdminController) Riders(c *gin.Context) {
	type row struct {
		ID uint `json:"id"`
		UserID uint `json:"userId"`
		VehiclePlate string `json:"vehiclePlate"`
		RiderStatusID uint `json:"riderStatusId"`
	}
	var items []row
	if err := ac.DB.Model(&entity.Rider{}).
		Select("id, user_id, vehicle_plate, rider_status_id").
		Order("id DESC").Limit(100).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items})
}

// โปรโมชัน
func getUintFromCtx(c *gin.Context, keys ...string) (uint, bool) {
	for _, k := range keys {
		if v, ok := c.Get(k); ok {
			switch x := v.(type) {
			case uint:
				return x, true
			case int:
				return uint(x), true
			case int64:
				return uint(x), true
			case float64:
				return uint(x), true
			case string:
				if u, err := strconv.ParseUint(x, 10, 64); err == nil {
					return uint(u), true
				}
			}
		}
	}
	return 0, false
}

func ensureAdmin(c *gin.Context) (uint, bool) {
	// role
	if roleVal, has := c.Get("role"); has {
		if roleStr, _ := roleVal.(string); roleStr != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return 0, false
		}
	} else {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return 0, false
	}

	// admin id
	adminID, ok := getUintFromCtx(c, "userId", "id", "userID")
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Admin ID not found in token"})
		return 0, false
	}
	return adminID, true
}
func parseDateFlexible(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02",
		"2006-01-02T15:04:05.000Z",
		"02/01/2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("invalid time format")
}








func (ac *AdminController) Promotions(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}

	// ดึงจากโมเดลจริง + Preload ก่อน แล้ว map เป็น response กะทัดรัด
	var promos []entity.Promotion
	if err := ac.DB.
		Preload("PromoType").
		Order("id DESC").
		Limit(100).
		Find(&promos).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type row struct {
		ID          uint              `json:"id"`
		PromoCode   string            `json:"promoCode"`
		PromoDetail string            `json:"promoDetail"`
		Values      uint              `json:"values"`
		MinOrder    int64             `json:"minOrder"`
		StartAt     *time.Time        `json:"startAt,omitempty"`
		EndAt       *time.Time        `json:"endAt,omitempty"`
		PromoTypeID uint              `json:"promoTypeId"`
		PromoType   *entity.PromoType `json:"promoType,omitempty"`
		AdminID     uint              `json:"adminId"`
	}

	items := make([]row, 0, len(promos))
	for _, p := range promos {
		pt := p.PromoType // copy
		items = append(items, row{
			ID:          p.ID,
			PromoCode:   p.PromoCode,
			PromoDetail: p.PromoDetail,
			Values:      p.Values,
			MinOrder:    p.MinOrder,
			StartAt:     p.StartAt,
			EndAt:       p.EndAt,
			PromoTypeID: p.PromoTypeID,
			PromoType:   &pt,
			AdminID:     p.AdminID,
		})
	}

	c.JSON(http.StatusOK, gin.H{"items": items})
}

// -------- Request Struct --------

type CreatePromotionReq struct {
	PromoCode   string `json:"promoCode" binding:"required"`
	PromoDetail string `json:"promoDetail" binding:"required"`
	Values      uint   `json:"values" binding:"required,min=1"`
	MinOrder    int64  `json:"minOrder" binding:"required,min=0"`
	StartAt     string `json:"startAt" binding:"required"`
	EndAt       string `json:"endAt" binding:"required"`
	PromoTypeID uint   `json:"promoTypeId" binding:"required"`
}

// POST /admin/promotion
func (ac *AdminController) CreatePromotion(c *gin.Context) {
	adminID, ok := ensureAdmin(c)
	if !ok {
		return
	}

	var req CreatePromotionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ถ้าเป็นเปอร์เซ็นต์ ต้อง 1-100
	if req.PromoTypeID == 2 && (req.Values < 1 || req.Values > 100) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"error": "Value for percentage promo must be between 1 and 100",
		})
		return
	}

	st, err := parseDateFlexible(req.StartAt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid startAt"})
		return
	}
	et, err := parseDateFlexible(req.EndAt)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid endAt"})
		return
	}

	p := entity.Promotion{
		PromoCode:   req.PromoCode,
		PromoDetail: req.PromoDetail,
		Values:      req.Values,
		MinOrder:    req.MinOrder,
		PromoTypeID: req.PromoTypeID,
		StartAt:     st,
		EndAt:       et,
		AdminID:     adminID, // มาจากโทเคน ไม่รับจาก frontend
	}

	if err := ac.DB.Create(&p).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": p.ID})
}

// PUT /admin/promotion/:id
type PromotionUpdateReq struct {
	PromoCode   *string `json:"promoCode"`
	PromoDetail *string `json:"promoDetail"`
	Values      *uint   `json:"values"`
	MinOrder    *int64  `json:"minOrder"`
	StartAt     *string `json:"startAt"`
	EndAt       *string `json:"endAt"`
	PromoTypeID *uint   `json:"promoTypeId"`
}

func (ac *AdminController) UpdatePromotion(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}

	id := c.Param("id")
	promoID, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid promotion ID"})
		return
	}

	var req PromotionUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var promotion entity.Promotion
	if err := ac.DB.First(&promotion, promoID).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Promotion not found"})
		return
	}

	if req.PromoCode != nil {
		promotion.PromoCode = *req.PromoCode
	}
	if req.PromoDetail != nil {
		promotion.PromoDetail = *req.PromoDetail
	}
	if req.MinOrder != nil {
		promotion.MinOrder = *req.MinOrder
	}
	if req.PromoTypeID != nil {
		promotion.PromoTypeID = *req.PromoTypeID
		if *req.PromoTypeID == 2 && req.Values != nil && (*req.Values < 1 || *req.Values > 100) {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error": "Value for percentage promo must be between 1 and 100",
			})
			return
		}
	}
	if req.Values != nil {
		promotion.Values = *req.Values
	}
	if req.StartAt != nil {
		st, err := parseDateFlexible(*req.StartAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid startAt"})
			return
		}
		promotion.StartAt = st
	}
	if req.EndAt != nil {
		et, err := parseDateFlexible(*req.EndAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid endAt"})
			return
		}
		promotion.EndAt = et
	}

	if err := ac.DB.Save(&promotion).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promotion updated successfully"})
}

// DELETE /admin/promotion/:id
// DELETE /admin/promotion/:id
func (ac *AdminController) DeletePromotion(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}

	id := c.Param("id")
	promoID, err := strconv.Atoi(id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid promotion ID"})
		return
	}

	// ตรวจว่ามีอยู่จริงก่อน
	var promotion entity.Promotion
	if err := ac.DB.First(&promotion, promoID).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Promotion not found"})
		return
	}

	// ใช้ทรานแซกชันเพื่อความปลอดภัย
	if err := ac.DB.Transaction(func(tx *gorm.DB) error {
		// (เผื่อ schema ยังไม่ได้ทำงาน CASCADE) ลบความสัมพันธ์ใน user_promotions แบบ hard ก่อน
		if err := tx.Unscoped().
			Where("promotion_id = ?", promotion.ID).
			Delete(&entity.UserPromotion{}).Error; err != nil {
			return err
		}

		// ลบโปรโมชั่นแบบ hard delete จริง ๆ
		if err := tx.Unscoped().Delete(&promotion).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promotion deleted successfully (hard delete)"})
}
