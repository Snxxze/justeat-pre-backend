package controllers

import (
	"backend/entity"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type MenuController struct {
	DB *gorm.DB
}

func NewMenuController(db *gorm.DB) *MenuController {
	return &MenuController{DB: db}
}

// GET /restaurants/:id/menus
func (ctl *MenuController) ListByRestaurant(c *gin.Context) {
	restID, _ := strconv.Atoi(c.Param("id"))

	var menus []entity.Menu
	if err := ctl.DB.
		Preload("MenuType").
		Preload("MenuStatus").
		Where("restaurant_id = ?", uint(restID)).
		Find(&menus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": menus})
}

// GET /menus/:id
func (ctl *MenuController) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var menu entity.Menu
	if err := ctl.DB.
		Preload("MenuType").
		Preload("MenuStatus").
		First(&menu, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "menu not found"})
		return
	}
	c.JSON(http.StatusOK, menu)
}

// POST /owner/restaurants/:id/menus
func (ctl *MenuController) Create(c *gin.Context) {
	restID, _ := strconv.Atoi(c.Param("id"))

	var req entity.Menu
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.RestaurantID = uint(restID)

	if err := ctl.DB.Create(&req).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, req)
}

// PATCH /owner/menus/:id
func (ctl *MenuController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req entity.Menu
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uint(id)

	fields := map[string]interface{}{
		"name":          req.Name,
		"detail":        req.Detail,
		"price":         req.Price,
		"image":         req.Image,
		"menu_type_id":  req.MenuTypeID,
		"menu_status_id": req.MenuStatusID,
	}

	if err := ctl.DB.Model(&entity.Menu{}).
		Where("id = ?", req.ID).
		Updates(fields).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

// DELETE /owner/menus/:id
func (ctl *MenuController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	if err := ctl.DB.Delete(&entity.Menu{}, uint(id)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "menu deleted"})
}

// PATCH /owner/menus/:id/status
func (ctl *MenuController) UpdateStatus(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	var req struct {
		MenuStatusID uint `json:"menuStatusId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctl.DB.Model(&entity.Menu{}).
		Where("id = ?", uint(id)).
		Update("menu_status_id", req.MenuStatusID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "menu status updated"})
}
