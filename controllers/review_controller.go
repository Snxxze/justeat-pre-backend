package controllers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend/entity"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ReviewController struct{ DB *gorm.DB }

func NewReviewController(db *gorm.DB) *ReviewController { return &ReviewController{DB: db} }

// ===== utils =====

// ดึง userId จาก context ตรง ๆ (middleware ต้อง c.Set("userId", <uint>) ไว้แล้ว)
func mustUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("userId")
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok && id != 0
}

func now() time.Time { return time.Now().UTC() }

// ระบุสถานะออร์เดอร์ที่ "รีวิวได้" ด้วย ID (ปรับตามข้อมูลจริงของคุณ)
var reviewableStatusIDs = map[uint]struct{}{
	// ตัวอย่าง: Completed=3, Delivered=5
	// 3: {}, 5: {},
}

// ===== DTO =====

type CreateReviewReq struct {
	OrderID  uint   `json:"orderId" binding:"required"`
	Rating   int    `json:"rating"  binding:"required,min=1,max=5"`
	Comments string `json:"comments"`
}

type UpdateReviewReq struct {
	Rating   *int   `json:"rating" binding:"omitempty,min=1,max=5"`
	Comments string `json:"comments"`
}

// ===== Presenter =====

func (rc *ReviewController) presentReview(r entity.Review) gin.H {
	var user any = nil
	if r.UserID != 0 {
		// ถ้ายังไม่ได้ preload → fallback แบบ select เฉพาะที่ใช้
		if r.User.ID == 0 {
			var u entity.User
			_ = rc.DB.Select("id, first_name, last_name").First(&u, r.UserID).Error
			r.User = u
		}
		if r.User.ID != 0 {
			user = gin.H{
				"id":        r.User.ID,
				"firstName": r.User.FirstName,
				"lastName":  r.User.LastName,
			}
		}
	}

	return gin.H{
		"id":         r.ID,
		"rating":     r.Rating,
		"comments":   r.Comments,
		"reviewDate": r.ReviewDate,
		"user":       user,
	}
}

// ===== Handlers =====

// POST /reviews (Protected) — 1 review ต่อ 1 order (upsert ด้วย order_id)
func (rc *ReviewController) Create(c *gin.Context) {
	uid, ok := mustUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}

	var req CreateReviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// 1) ตรวจออร์เดอร์เป็นของ user (ดึงร้านไปด้วยเพื่อกัน owner)
	var ord entity.Order
	if err := rc.DB.
		Select("id, user_id, restaurant_id, order_status_id").
		Where("id = ? AND user_id = ?", req.OrderID, uid).
		First(&ord).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "order not found or not belong to user"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		}
		return
	}

	// 2) สถานะต้องรีวิวได้ (ตาม ID ที่กำหนด)
	if _, ok := reviewableStatusIDs[ord.OrderStatusID]; !ok && len(reviewableStatusIDs) > 0 {
		// ถ้า map ว่าง จะข้ามการบังคับ (เผื่อ DEV); ถ้าไม่ว่าง → ต้องอยู่ในลิสต์
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": "order is not in a reviewable status"})
		return
	}

	// 3) กัน owner รีวิวร้านตัวเอง
	var rs entity.Restaurant
	if err := rc.DB.Select("id, user_id").First(&rs, ord.RestaurantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}
	if rs.UserID == uid {
		c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "owners cannot review their own restaurant"})
		return
	}

	// 4) Upsert ด้วย unique(order_id)
	rev := entity.Review{
		Rating:       req.Rating,
		Comments:     strings.TrimSpace(req.Comments),
		ReviewDate:   now(),
		UserID:       uid,
		RestaurantID: ord.RestaurantID,
		OrderID:      req.OrderID,
	}
	if err := rc.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "order_id"}}, // ชนด้วย order_id → update
		DoUpdates: clause.AssignmentColumns([]string{"rating", "comments", "review_date"}),
	}).Create(&rev).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ok": false, "error": err.Error()})
		return
	}

	// โหลด user เฉพาะฟิลด์ที่ใช้แสดง
	_ = rc.DB.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, first_name, last_name")
	}).First(&rev, "order_id = ?", req.OrderID).Error

	c.JSON(http.StatusOK, gin.H{"ok": true, "review": rc.presentReview(rev)})
}

// GET /restaurants/:id/reviews?limit=20&offset=0[&rating=4]
// ส่ง { rows, avg, total } — โดย total = จำนวนหลังกรอง (สอดคล้องกับ filter/pagination)
func (rc *ReviewController) ListForRestaurant(c *gin.Context) {
	rid, err := strconv.Atoi(c.Param("id"))
	if err != nil || rid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid restaurant id"})
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

	// filter ตามดาว (ถ้ามี)
	var ratingFilter *int
	if rs := c.Query("rating"); rs != "" {
		if v, err := strconv.Atoi(rs); err == nil && v >= 1 && v <= 5 {
			ratingFilter = &v
		}
	}

	// total (ตาม filter) — ใช้กับ FE paginate
	var total int64
	ct := rc.DB.Model(&entity.Review{}).Where("restaurant_id = ?", rid)
	if ratingFilter != nil {
		ct = ct.Where("rating = ?", *ratingFilter)
	}
	if err := ct.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "count failed"})
		return
	}

	// rows (เลือกคอลัมน์ user ให้เล็กลง)
	var reviews []entity.Review
	q := rc.DB.
		Where("restaurant_id = ?", rid).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		})
	if ratingFilter != nil {
		q = q.Where("rating = ?", *ratingFilter)
	}
	if err := q.Order("review_date DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// avg ของ "ทั้งร้าน" (ไม่ขึ้นกับ filter) + COALESCE กัน NULL
	var avgRow struct{ Avg float64 }
	if err := rc.DB.Model(&entity.Review{}).
		Where("restaurant_id = ?", rid).
		Select("COALESCE(AVG(rating), 0) AS avg").
		Scan(&avgRow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "aggregate failed"})
		return
	}

	// map → ฝัง user
	items := make([]gin.H, 0, len(reviews))
	for _, r := range reviews {
		items = append(items, rc.presentReview(r))
	}

	c.JSON(http.StatusOK, gin.H{
		"rows":  items,
		"avg":   avgRow.Avg, // ค่าเฉลี่ยทั้งร้าน
		"total": total,      // จำนวนรีวิวที่ตรงกับ filter (สำหรับ paginate)
	})
}

// GET /profile/reviews (Protected)
func (rc *ReviewController) ListForMe(c *gin.Context) {
	uid, ok := mustUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}

	// โหมดสรุปแยกร้าน: /profile/reviews?group=restaurant
	if c.Query("group") == "restaurant" {
		type Row struct {
			RestaurantID   uint
			Name           string
			Count          int64
			Avg            float64
			LastReviewDate time.Time
		}
		var rows []Row
		if err := rc.DB.Table("reviews r").
			Select(`r.restaurant_id AS restaurant_id, rs.name AS name,
			        COUNT(*) AS count, AVG(r.rating) AS avg,
			        MAX(r.review_date) AS last_review_date`).
			Joins("JOIN restaurants rs ON rs.id = r.restaurant_id").
			Where("r.user_id = ?", uid).
			Group("r.restaurant_id, rs.name").
			Order("last_review_date DESC").
			Scan(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
			return
		}

		items := make([]gin.H, 0, len(rows))
		for _, v := range rows {
			items = append(items, gin.H{
				"restaurant": gin.H{"id": v.RestaurantID, "name": v.Name},
				"count":      v.Count,
				"avgRating":  v.Avg,
				"lastReview": v.LastReviewDate,
			})
		}
		c.JSON(http.StatusOK, gin.H{"ok": true, "items": items})
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
	if err := rc.DB.
		Preload("Restaurant", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, name")
		}).
		Where("user_id = ?", uid).
		Order("review_date DESC").
		Limit(limit).Offset(offset).
		Find(&reviews).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		return
	}

	items := make([]gin.H, 0, len(reviews))
	for _, r := range reviews {
		var restaurant any = nil
		if r.Restaurant.ID != 0 {
			restaurant = gin.H{
				"id":   r.Restaurant.ID,
				"name": r.Restaurant.Name,
			}
		}
		items = append(items, gin.H{
			"id":         r.ID,
			"rating":     r.Rating,
			"comments":   r.Comments,
			"reviewDate": r.ReviewDate,
			"restaurant": restaurant,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"ok":    true,
		"items": items,
		"meta":  gin.H{"limit": limit, "offset": offset},
	})
}

// GET /reviews/:id (Protected) ← เจ้าของรีวิวเท่านั้น
func (rc *ReviewController) DetailForMe(c *gin.Context) {
	uid, ok := mustUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "unauthorized"})
		return
	}
	id, _ := strconv.Atoi(c.Param("id"))

	var rev entity.Review
	if err := rc.DB.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, first_name, last_name")
		}).
		Where("id = ? AND user_id = ?", id, uid).
		First(&rev).Error; err != nil {

		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"ok": false, "error": "review not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"ok": false, "error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "review": rc.presentReview(rev)})
}
