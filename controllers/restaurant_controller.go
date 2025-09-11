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

// ====== Public: ดูร้านทั้งหมด (รองรับ filter ด้วย categoryId) ======
func (ctl *RestaurantController) List(c *gin.Context) {
	categoryId := c.Query("categoryId")
	statusId := c.Query("statusId")

	rests, err := ctl.Service.ListFiltered(categoryId, statusId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []RestaurantResponse
	for _, r := range rests {
		resp = append(resp, mapToRestaurantResponse(&r))
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
  id, err := strconv.Atoi(c.Param("id"))
  if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"}); return }

  // ⬇️ ดึง userID จาก middleware (ปรับตามที่คุณเก็บใน context)
  uidAny, ok := c.Get("userId")
  if !ok { c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"}); return }
  userID := uidAny.(uint)

  // เช็คว่า restaurant นี้เป็นของ userID จริง
  var count int64
  if err := ctl.Service.Repo.DB.Model(&entity.Restaurant{}).
    Where("id = ? AND user_id = ?", id, userID).
    Count(&count).Error; err != nil {
    c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
    return
  }
  if count == 0 {
    c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
    return
  }

  // ====== bind + build updates เหมือนเดิม ======
  var in struct {
    Name                 *string `json:"name"`
    Address              *string `json:"address"`
    Description          *string `json:"description"`
    PictureBase64        *string `json:"pictureBase64"`
    OpeningTime          *string `json:"openingTime"`
    ClosingTime          *string `json:"closingTime"`
    RestaurantCategoryID *uint   `json:"restaurantCategoryId"`
    RestaurantStatusID   *uint   `json:"restaurantStatusId"`
  }
  if err := c.ShouldBindJSON(&in); err != nil {
    c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return
  }

  updates := map[string]interface{}{}
  if in.Name != nil            { updates["name"] = *in.Name }
  if in.Address != nil         { updates["address"] = *in.Address }
  if in.Description != nil     { updates["description"] = *in.Description }
  if in.PictureBase64 != nil   { updates["picture_base64"] = *in.PictureBase64 }
  if in.OpeningTime != nil     { updates["opening_time"] = *in.OpeningTime }
  if in.ClosingTime != nil     { updates["closing_time"] = *in.ClosingTime }
  if in.RestaurantCategoryID != nil { updates["restaurant_category_id"] = *in.RestaurantCategoryID }
  if in.RestaurantStatusID != nil   { updates["restaurant_status_id"] = *in.RestaurantStatusID }

  if len(updates) == 0 {
    c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
    return
  }

  if err := ctl.Service.Repo.DB.
    Model(&entity.Restaurant{}).
    Where("id = ? AND user_id = ?", id, userID).
    Updates(updates).Error; err != nil {
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
