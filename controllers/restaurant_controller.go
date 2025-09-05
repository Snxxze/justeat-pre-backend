// controllers/restaurant_controller.go
package controllers

import (
	"backend/entity"
	"backend/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type RestaurantController struct {
	Service *services.RestaurantService
}

func NewRestaurantController(s *services.RestaurantService) *RestaurantController {
	return &RestaurantController{Service: s}
}

// POST /partner/restaurant
func (rc *RestaurantController) CreateRestaurant(c *gin.Context) {
	var req entity.Restaurant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ถ้า frontend ส่ง picture base64 มาใน JSON
	base64Img := ""
	if req.Picture != "" {
		base64Img = req.Picture
		req.Picture = "" // เคลียร์ก่อนเพื่อกัน base64 ยาว ๆ ไปติดใน DB
	}

	id, err := rc.Service.CreateRestaurant(&req, base64Img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// GET /partner/restaurant/menu
func (rc *RestaurantController) Menus(c *gin.Context) {
	restID, _ := strconv.Atoi(c.DefaultQuery("restaurantId", "0"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	items, total, err := rc.Service.GetMenus(uint(restID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "page": page, "limit": limit, "total": total})
}

// POST /partner/restaurant/menu
func (rc *RestaurantController) CreateMenu(c *gin.Context) {
	var req entity.Menu
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}

	base64Img := c.PostForm("picture") // หรืออ่านจาก req.Picture ถ้าเป็น JSON
	id, err := rc.Service.CreateMenu(&req, base64Img)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

// PATCH /partner/restaurant/menu/:id
func (rc *RestaurantController) UpdateMenu(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	var base64Img *string
	if pic, ok := req["picture"].(string); ok && pic != "" {
		base64Img = &pic
	}
	delete(req, "picture")

	if err := rc.Service.UpdateMenu(uint(id), req, base64Img); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": 1})
}

// GET /partner/restaurant/dashboard
func (rc *RestaurantController) Dashboard(c *gin.Context) {
	restID, _ := strconv.Atoi(c.DefaultQuery("restaurantId", "0"))
	orders, revenue, err := rc.Service.Dashboard(uint(restID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()}); return
	}
	c.JSON(http.StatusOK, gin.H{"ordersToday": orders, "revenue": revenue})
}
