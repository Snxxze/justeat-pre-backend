// controllers/review_controller.go
package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"backend/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReviewController struct{ DB *gorm.DB }

func NewReviewController(db *gorm.DB) *ReviewController { return &ReviewController{DB: db} }

// ใช้ orderId หลอกสำหรับโหมดเร็ว (ยังไม่มี order จริง)
const sentinelBase uint = 900_000_000

// ===== utils =====

func getUserIDFromCtx(c *gin.Context) (uint, bool) {
	// รองรับหลายคีย์ที่มักใช้กัน
	keys := []string{"userID", "userId", "uid", "user_id", "id", "sub"}
	for _, k := range keys {
		if v, ok := c.Get(k); ok {
			switch x := v.(type) {
			case uint:
				return x, true
			case int:
				if x >= 0 { return uint(x), true }
			case int64:
				if x >= 0 { return uint(x), true }
			case float64:
				if x >= 0 { return uint(x), true }
			case string:
				if n, err := strconv.ParseUint(x, 10, 64); err == nil {
					return uint(n), true
				}
			}
		}
	}

	// เผื่อ middleware ยัด claims ทั้งก้อน
	if cl, ok := c.Get("claims"); ok {
		if m, ok := cl.(map[string]any); ok {
			if v, ok := m["userId"]; ok {
				switch t := v.(type) {
				case float64:
					if t >= 0 { return uint(t), true }
				case string:
					if n, err := strconv.ParseUint(t, 10, 64); err == nil { return uint(n), true }
				}
			}
			if v, ok := m["user_id"]; ok {
				switch t := v.(type) {
				case float64:
					if t >= 0 { return uint(t), true }
				case string:
					if n, err := strconv.ParseUint(t, 10, 64); err == nil { return uint(n), true }
				}
			}
			if s, ok := m["sub"].(string); ok {
				if n, err := strconv.ParseUint(s, 10, 64); err == nil { return uint(n), true }
			}
		}
	}

	return 0, false
}


func now() time.Time { return time.Now() }

// ===== DTO =====

type CreateReviewReq struct {
	OrderID      uint   `json:"orderId"`                // ถ้ามี: ผูกกับ order จริง (ของ user เท่านั้น)
	RestaurantID uint   `json:"restaurantId,omitempty"` // ถ้าไม่ส่ง orderId ต้องส่งอันนี้
	Rating       int    `json:"rating" binding:"required,min=1,max=5"`
	Comments     string `json:"comments"`
}

type UpdateReviewReq struct {
	Rating   *int   `json:"rating" binding:"omitempty,min=1,max=5"`
	Comments string `json:"comments"`
}

// ===== Handlers =====

// POST /reviews (Protected)
// - ถ้ามี orderId → ตรวจว่าเป็นของ user → ผูกรีวิวกับร้านจากออเดอร์นั้น (อัปเดตถ้าเคยรีวิวร้านนี้แล้ว)
// - ถ้าไม่มี orderId แต่มี restaurantId → โหมดเร็ว (upsert ต่อ user+restaurant)
func (rc *ReviewController) Create(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}

	var req CreateReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	var restaurantID uint
	var orderID uint

	if req.OrderID > 0 {
		// ตรวจออเดอร์ของ user
		var ord entity.Order
		if err := rc.DB.Where("id = ? AND user_id = ?", req.OrderID, uid).First(&ord).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "order not found or not belong to user"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			}
			return
		}
		restaurantID = ord.RestaurantID
		orderID = req.OrderID

		// เคยรีวิวร้านนี้แล้ว? → อัปเดตแทน
		var exist entity.Review
		if err := rc.DB.Where("user_id = ? AND restaurant_id = ?", uid, restaurantID).
			Order("review_date DESC").First(&exist).Error; err == nil {
			exist.Rating = req.Rating
			exist.Comments = req.Comments
			exist.ReviewDate = now()
			exist.OrderID = orderID // ถ้าเคยใช้ sentinel ให้โยกมาเป็น order จริง
			if err := rc.DB.Save(&exist).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "review": exist})
			return
		}

		// ยังไม่เคย → สร้างใหม่
		rev := entity.Review{
			Rating:       req.Rating,
			Comments:     req.Comments,
			ReviewDate:   now(),
			UserID:       uid,
			RestaurantID: restaurantID,
			OrderID:      orderID,
		}
		if err := rc.DB.Create(&rev).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"ok": true, "review": rev})
		return
	}

	// ไม่มี orderId → ต้องมี restaurantId
	if req.RestaurantID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "either orderId or restaurantId is required"})
		return
	}
	restaurantID = req.RestaurantID
	orderID = sentinelBase + restaurantID // sentinel ช่วยไม่ให้ชน unique (user_id, order_id)

	// upsert ต่อร้าน
	var exist entity.Review
	if err := rc.DB.Where("user_id = ? AND restaurant_id = ?", uid, restaurantID).
		Order("review_date DESC").First(&exist).Error; err == nil {
		exist.Rating = req.Rating
		exist.Comments = req.Comments
		exist.ReviewDate = now()
		if exist.OrderID == 0 {
			exist.OrderID = orderID
		}
		if err := rc.DB.Save(&exist).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "review": exist})
		return
	}

	rev := entity.Review{
		Rating:       req.Rating,
		Comments:     req.Comments,
		ReviewDate:   now(),
		UserID:       uid,
		RestaurantID: restaurantID,
		OrderID:      orderID,
	}
	if err := rc.DB.Create(&rev).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ok": true, "review": rev})
}

// GET /restaurants/:id/reviews (Public)  ← ชื่อตรงกับ routes: ListForRestaurant
func (rc *ReviewController) ListForRestaurant(c *gin.Context) {
	rid, _ := strconv.Atoi(c.Param("id"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var reviews []entity.Review
	if err := rc.DB.Where("restaurant_id = ?", rid).
		Order("review_date DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// สรุปเร็ว ๆ
	type agg struct{ Avg float64; Count int64 }
	var a agg
	_ = rc.DB.Model(&entity.Review{}).
		Where("restaurant_id = ?", rid).
		Select("AVG(rating) AS avg, COUNT(*) AS count").
		Scan(&a).Error

	c.JSON(http.StatusOK, gin.H{
		"ok":        true,
		"items":     reviews,
		"meta":      gin.H{"limit": limit, "offset": offset},
		"aggregate": gin.H{"avgRating": a.Avg, "total": a.Count},
	})
}

// GET /profile/reviews (Protected) ← ชื่อตรงกับ routes: ListForMe
func (rc *ReviewController) ListForMe(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var reviews []entity.Review
	if err := rc.DB.Where("user_id = ?", uid).
		Order("review_date DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"items": reviews,
		"meta":  gin.H{"limit": limit, "offset": offset},
	})
}

// GET /reviews/:id (Protected) ← ชื่อตรงกับ routes: DetailForMe (owner only)
func (rc *ReviewController) DetailForMe(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))

	var rev entity.Review
	if err := rc.DB.Where("id = ? AND user_id = ?", id, uid).First(&rev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "review not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "review": rev})
}

// (ถ้าต้องการใช้ต่อภายหน้า) PUT /reviews/:id, DELETE /reviews/:id สามารถเพิ่มเมธอด Update/Delete เหมือนเดิมได้
