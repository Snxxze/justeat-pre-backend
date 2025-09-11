package controllers

import (
	"errors"
	"net/http"
	"strconv"

	"backend/services"
	"github.com/gin-gonic/gin"
)

type UserPromotionController struct {
	userPromotionService *services.UserPromotionService
}

func NewUserPromotionController(s *services.UserPromotionService) *UserPromotionController {
	return &UserPromotionController{userPromotionService: s}
}

// ---------- helpers ----------
func getUserID(c *gin.Context) (uint, bool) {
	// รองรับทั้ง "userId" และ "user_id"
	if v, ok := c.Get("userId"); ok {
		if id, ok := v.(uint); ok {
			return id, true
		}
	}
	if v, ok := c.Get("user_id"); ok {
		if id, ok := v.(uint); ok {
			return id, true
		}
	}
	c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
	return 0, false
}

// ---------- POST /user/promotions |  POST /user/promotions/:id ----------
func (ctrl *UserPromotionController) SavePromotion(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	// 1) จาก URL param (/user/promotions/:id)
	var promoID uint
	if idStr := c.Param("id"); idStr != "" {
		n, err := strconv.Atoi(idStr)
		if err != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promotion id"})
			return
		}
		promoID = uint(n)
	}

	// 2) จาก JSON body
	if promoID == 0 {
		var body struct {
			PromoId     uint `json:"promoId"`
			PromotionId uint `json:"promotionId"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
			return
		}
		promoID = body.PromoId
		if promoID == 0 {
			promoID = body.PromotionId
		}
	}

	if promoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing promoId / promotionId"})
		return
	}

	// เรียก service
	if err := ctrl.userPromotionService.SavePromotion(userID, promoID); err != nil {
		switch {
		case errors.Is(err, services.ErrAlreadySaved):
			c.JSON(http.StatusConflict, gin.H{"error": "promotion already saved"})
		case errors.Is(err, services.ErrPromotionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "promotion not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save promotion"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "promotion saved"})
}

// ---------- POST /user/promotions/:id/use  |  POST /user/promotions/use ----------
func (ctrl *UserPromotionController) UsePromotion(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	// รับ id จาก param หรือ body
	var promoID uint
	if idStr := c.Param("id"); idStr != "" {
		n, err := strconv.Atoi(idStr)
		if err != nil || n <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid promotion id"})
			return
		}
		promoID = uint(n)
	}
	if promoID == 0 {
		var body struct {
			PromoId     uint `json:"promoId"`
			PromotionId uint `json:"promotionId"`
		}
		if err := c.ShouldBindJSON(&body); err == nil {
			promoID = body.PromoId
			if promoID == 0 {
				promoID = body.PromotionId
			}
		}
	}
	if promoID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing promoId / promotionId"})
		return
	}

	if err := ctrl.userPromotionService.UsePromotion(userID, promoID); err != nil {
		switch {
		case errors.Is(err, services.ErrUserPromotionNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "user promotion not found"})
		case errors.Is(err, services.ErrAlreadyUsed):
			c.JSON(http.StatusConflict, gin.H{"error": "promotion already used"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to use promotion"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "promotion used"})
}

// ---------- GET /user/promotions ----------
func (ctrl *UserPromotionController) List(c *gin.Context) {
	userID, ok := getUserID(c)
	if !ok {
		return
	}

	// แนะนำให้ service ทำ DB.Preload("Promotion") ด้วย
	rows, err := ctrl.userPromotionService.List(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list user promotions"})
		return
	}
	c.JSON(http.StatusOK, rows)
}
