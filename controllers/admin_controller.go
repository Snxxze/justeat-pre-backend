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
func (ac *AdminController) Promotions(c *gin.Context) {
	type row struct {
		ID uint `json:"id"`
		PromoCode string `json:"promoCode"`
		PromoTypeID uint `json:"promoTypeId"`
		MinOrder int64 `json:"minOrder"`
		StartAt *time.Time `json:"startAt,omitempty"`
		EndAt *time.Time `json:"endAt,omitempty"`
	}
	var items []row
	if err := ac.DB.Model(&entity.Promotion{}).
		Select("id, promo_code, promo_type_id, min_order, start_at, end_at").
		Order("id DESC").Limit(100).Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items})
}

type CreatePromotionReq struct {
	PromoCode   string `json:"promoCode" binding:"required"`
	PromoDetail string `json:"promoDetail"`
	PromoTypeID uint   `json:"promoTypeId" binding:"required"`
	IsValues    bool   `json:"isValues"`
	MinOrder    int64  `json:"minOrder"`
	// รับได้ทั้ง "2006-01-02" หรือ RFC3339
	StartAt     string `json:"startAt"`
	EndAt       string `json:"endAt"`
	AdminID     uint   `json:"adminId"`
}

func parseDateFlexible(s string) (*time.Time, error) {
	if s == "" { return nil, nil }
	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil { return &t, nil }
	}
	return nil, fmt.Errorf("invalid time format")
}

func (ac *AdminController) CreatePromotion(c *gin.Context) {
	var req CreatePromotionReq
	if err := c.ShouldBindJSON(&req); err != nil { resp.BadRequest(c, err.Error()); return }
	st, err := parseDateFlexible(req.StartAt); if err != nil { resp.BadRequest(c, "invalid startAt"); return }
	et, err := parseDateFlexible(req.EndAt);   if err != nil { resp.BadRequest(c, "invalid endAt"); return }

	p := entity.Promotion{
		PromoCode: req.PromoCode, PromoDetail: req.PromoDetail,
		IsValues: req.IsValues, MinOrder: req.MinOrder,
		PromoTypeID: req.PromoTypeID,
		StartAt: st, EndAt: et,
		AdminID: req.AdminID,
	}
	if err := ac.DB.Create(&p).Error; err != nil { resp.ServerError(c, err); return }
	resp.Created(c, gin.H{"id": p.ID})
}
