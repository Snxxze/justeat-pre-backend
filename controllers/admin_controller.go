package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"backend/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AdminController struct {
	DB *gorm.DB
}

func NewAdminController(db *gorm.DB) *AdminController {
	return &AdminController{DB: db}
}

// =========================
// Helpers
// =========================

// แปลงค่าจาก gin.Context เป็น uint (ลองหลายชนิด/หลายคีย์)
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

// ensureAdmin: ต้องมี role=admin และคืน adminID (จากโทเคน) ถ้าไม่ผ่านจะ Abort และคืน false
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

// ฟังก์ชัน parse เวลา ให้ยืดหยุ่นกับรูปแบบวันที่/เวลา
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

// =========================
// Dashboard / Restaurants / Reports / Riders
// =========================

func (ac *AdminController) Dashboard(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "admin dashboard"})
}

func (ac *AdminController) Restaurants(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": []any{}, "message": "admin restaurants"})
}

func (ac *AdminController) Reports(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"report": gin.H{}, "message": "admin reports"})
}

func (ac *AdminController) Riders(c *gin.Context) {
	if _, ok := ensureAdmin(c); !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": []any{}, "message": "admin riders"})
}

// =========================
// Promotions
// =========================

// GET /admin/promotion
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

	var promotion entity.Promotion
	if err := ac.DB.First(&promotion, promoID).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Promotion not found"})
		return
	}

	if err := ac.DB.Delete(&promotion).Error; err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Promotion deleted successfully"})
}
