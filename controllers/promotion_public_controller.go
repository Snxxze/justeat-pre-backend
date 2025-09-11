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

		// ปรับชื่อคอลัมน์ให้ตรง schema จริง (start_at/end_at หรือ startAt/endAt)
		if err := db.
			Where("(start_at IS NULL OR start_at <= ?) AND (end_at IS NULL OR end_at >= ?)", now, now).
			Order("id DESC").
			Find(&rows).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch promotions"})
			return
		}
		c.JSON(http.StatusOK, rows)
	}
}
