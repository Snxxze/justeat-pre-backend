// controllers/menu_controller.go
package controllers

import (
	"backend/entity"
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MenuController struct {
	Service *services.MenuService
}

func NewMenuController(s *services.MenuService) *MenuController {
	return &MenuController{Service: s}
}

// GET /restaurants/:id/menus
func (ctl *MenuController) ListByRestaurant(c *gin.Context) {
	restID, _ := strconv.Atoi(c.Param("id"))
	menus, err := ctl.Service.ListByRestaurant(uint(restID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": menus})
}

// GET /menus/:id
func (ctl *MenuController) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	menu, err := ctl.Service.Get(uint(id))
	if err != nil {
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

	if err := ctl.Service.Create(&req); err != nil {
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

	if err := ctl.Service.Update(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, req)
}

// DELETE /owner/menus/:id
func (ctl *MenuController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := ctl.Service.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "menu deleted"})
}
