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

// ====== Response DTO ======
type RestaurantResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	OpeningTime string `json:"openingTime"`
	ClosingTime string `json:"closingTime"`

	Category struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"category"`

	Status struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"status"`

	Owner struct {
		ID        uint   `json:"id"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	} `json:"owner"`
}

// ====== Public: ดูร้านทั้งหมด ======
func (ctl *RestaurantController) List(c *gin.Context) {
	rests, err := ctl.Service.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []RestaurantResponse
	for _, r := range rests {
		item := mapToRestaurantResponse(&r)
		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ====== Public: ดูร้านเดี่ยว ======
func (ctl *RestaurantController) Get(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	r, err := ctl.Service.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "restaurant not found"})
		return
	}

	resp := mapToRestaurantResponse(r)
	c.JSON(http.StatusOK, resp)
}

// ====== Owner: อัปเดตร้านของตัวเอง ======
func (ctl *RestaurantController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req entity.Restaurant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ID = uint(id)
	if err := ctl.Service.Update(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "restaurant updated"})
}

// ====== Helper ======
func mapToRestaurantResponse(r *entity.Restaurant) RestaurantResponse {
	item := RestaurantResponse{
		ID:          r.ID,
		Name:        r.Name,
		Address:     r.Address,
		Description: r.Description,
		Logo:        r.Picture,
		OpeningTime: r.OpeningTime,
		ClosingTime: r.ClosingTime,
	}
	item.Category.ID = r.RestaurantCategory.ID
	item.Category.Name = r.RestaurantCategory.CategoryName
	item.Status.ID = r.RestaurantStatus.ID
	item.Status.Name = r.RestaurantStatus.StatusName
	item.Owner.ID = r.User.ID
	item.Owner.FirstName = r.User.FirstName
	item.Owner.LastName = r.User.LastName
	item.Owner.Email = r.User.Email
	return item
}
