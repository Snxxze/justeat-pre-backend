package controllers

import (
	"net/http"
	"time"

	"backend/entity"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /promotions (public)
func ListActivePromotions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rows []entity.Promotion
		now := time.Now()

		q := db.
			Where("(start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", now, now)

		// filter ตาม code ถ้ามี query param
		if code := c.Query("code"); code != "" {
			q = q.Where("promo_code = ?", code)
		}

		if err := q.Order("id DESC").Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch promotions"})
			return
		}

		c.JSON(http.StatusOK, rows)
	}
}
