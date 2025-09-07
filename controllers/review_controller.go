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

// ใช้ orderId หลอกสำหรับโหมดเร็ว (ไม่มี order จริง)
// ควรตั้งให้ "สูงกว่า" id ออเดอร์จริงที่จะเกิดขึ้น เพื่อเลี่ยงชน
const sentinelBase uint = 900_000_000

// ===== utils =====

func getUserIDFromCtx(c *gin.Context) (uint, bool) {
	if v, ok := c.Get("userID"); ok {
		switch x := v.(type) {
		case uint:
			return x, true
		case int:
			if x >= 0 {
				return uint(x), true
			}
		case float64:
			if x >= 0 {
				return uint(x), true
			}
		}
	}
	return 0, false
}

func now() time.Time { return time.Now() }

// ===== DTO =====

type CreateReviewReq struct {
	OrderID      uint   `json:"orderId"`                // ส่งมาก็ผูกกับ order จริง
	RestaurantID uint   `json:"restaurantId,omitempty"` // ถ้าไม่ได้ส่ง orderId ต้องส่งอันนี้มาแทน
	Rating       int    `json:"rating" binding:"required,min=1,max=5"`
	Comments     string `json:"comments"`
}

type UpdateReviewReq struct {
	Rating   *int   `json:"rating" binding:"omitempty,min=1,max=5"`
	Comments string `json:"comments"`
}

// ===== Handlers =====

// POST /reviews  (ต้อง Auth)
// - ถ้ามี orderId: ตรวจว่าออเดอร์เป็นของ user แล้วสร้าง/อัปเดตรีวิวของร้านนั้น (อัปเดตแทนเพื่อกันซ้ำร้านเดิม)
// - ถ้าไม่มี orderId แต่มี restaurantId: ใช้โหมดเร็ว (bypass ในตัว) → upsert ต่อร้านต่อผู้ใช้
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
		// โหลดออเดอร์จริงของ user
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

		// ถ้าเคยมีรีวิวร้านนี้อยู่แล้ว (รวมถึงที่เคยสร้างแบบเร็วด้วย) → อัปเดตแทน
		var exist entity.Review
		if err := rc.DB.
			Where("user_id = ? AND restaurant_id = ?", uid, restaurantID).
			Order("review_date DESC").
			First(&exist).Error; err == nil {
			exist.Rating = req.Rating
			exist.Comments = req.Comments
			exist.ReviewDate = now()
			exist.OrderID = orderID // “โยก” จาก sentinel ไปเป็น order จริงด้วย
			if err := rc.DB.Save(&exist).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "review": exist})
			return
		}

		// ยังไม่เคยรีวิวร้านนี้เลย → สร้างใหม่
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
	orderID = sentinelBase + restaurantID // สร้าง id หลอก “ผูกต่อร้าน” เพื่อไม่ชน unique (user_id, order_id)

	// ถ้ามีรีวิวร้านนี้อยู่แล้ว → อัปเดต
	var exist entity.Review
	if err := rc.DB.
		Where("user_id = ? AND restaurant_id = ?", uid, restaurantID).
		Order("review_date DESC").
		First(&exist).Error; err == nil {
		exist.Rating = req.Rating
		exist.Comments = req.Comments
		exist.ReviewDate = now()
		// ถ้ารีวิวเดิมเป็น sentinel ก็รักษาไว้; ถ้าอยาก convert เป็น sentinel ทุกครั้งก็ตั้งค่าด้านล่าง
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

	// ยังไม่เคยรีวิว → สร้างใหม่ด้วย sentinel orderId
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

// GET /restaurants/:id/reviews
func (rc *ReviewController) ListByRestaurant(c *gin.Context) {
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

	// สรุปเบื้องต้น
	type agg struct {
		Avg   float64
		Count int64
	}
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

// GET /restaurants/:id/reviews/summary
func (rc *ReviewController) SummaryByRestaurant(c *gin.Context) {
	rid, _ := strconv.Atoi(c.Param("id"))

	type agg struct {
		Avg   float64
		Count int64
	}
	var a agg
	if err := rc.DB.Model(&entity.Review{}).
		Where("restaurant_id = ?", rid).
		Select("AVG(rating) AS avg, COUNT(*) AS count").
		Scan(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	type bucket struct {
		Rating int
		Total  int64
	}
	var b []bucket
	if err := rc.DB.Model(&entity.Review{}).
		Where("restaurant_id = ?", rid).
		Select("rating, COUNT(*) AS total").
		Group("rating").
		Scan(&b).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	dist := map[int]int64{1: 0, 2: 0, 3: 0, 4: 0, 5: 0}
	for _, x := range b {
		if x.Rating >= 1 && x.Rating <= 5 {
			dist[x.Rating] = x.Total
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":           true,
		"avg":          a.Avg,
		"total":        a.Count,
		"distribution": dist,
	})
}

// GET /orders/:id/review  (ต้อง Auth)
func (rc *ReviewController) GetMyReviewForOrder(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	oid, _ := strconv.Atoi(c.Param("id"))

	var rev entity.Review
	if err := rc.DB.Where("order_id = ? AND user_id = ?", oid, uid).First(&rev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "review not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "review": rev})
}

// PUT /reviews/:id  (ต้อง Auth)
func (rc *ReviewController) Update(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	rid, _ := strconv.Atoi(c.Param("id"))

	var body UpdateReviewReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	var rev entity.Review
	if err := rc.DB.Where("id = ? AND user_id = ?", rid, uid).First(&rev).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "review not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		}
		return
	}

	if body.Rating != nil {
		rev.Rating = *body.Rating
	}
	if body.Comments != "" || body.Comments == "" {
		rev.Comments = body.Comments
	}
	rev.ReviewDate = now()

	if err := rc.DB.Save(&rev).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "review": rev})
}

// DELETE /reviews/:id  (ต้อง Auth)
func (rc *ReviewController) Delete(c *gin.Context) {
	uid, ok := getUserIDFromCtx(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	rid, _ := strconv.Atoi(c.Param("id"))

	res := rc.DB.Where("id = ? AND user_id = ?", rid, uid).Delete(&entity.Review{})
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "review not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
