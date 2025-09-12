// controllers/restaurant_application_controller.go
package controllers

import (
	"backend/entity"
	"backend/utils"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RestaurantApplicationController struct {
	DB *gorm.DB
}

func NewRestaurantApplicationController(db *gorm.DB) *RestaurantApplicationController {
	return &RestaurantApplicationController{DB: db}
}

// ====== Request DTO ======
type ApplyRestaurantReq struct {
	Name                 string `json:"name" binding:"required"`
	Phone                string `json:"phone"`
	Address              string `json:"address"`
	Description          string `json:"description"`
	PictureBase64        string `json:"pictureBase64"`
	OpeningTime          string `json:"openingTime" binding:"required"`
	ClosingTime          string `json:"closingTime" binding:"required"`
	RestaurantCategoryID uint   `json:"restaurantCategoryId" binding:"required"`
}

// ====== Response DTO ======
type RestaurantApplicationResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Address     string `json:"address"`
	Phone       string `json:"phone"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	OpeningTime string `json:"openingTime"`
	ClosingTime string `json:"closingTime"`
	SubmittedAt string `json:"submittedAt"`

	RestaurantCategory struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	} `json:"restaurantCategory"`

	OwnerUser struct {
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Email       string `json:"email"`
		PhoneNumber string `json:"phoneNumber"`
	} `json:"ownerUser"`

	Status string `json:"status"`
}

type ApplyResponse struct {
	ID     uint   `json:"id"`
	Status string `json:"status"`
}

type ApproveResponse struct {
	ApplicationID uint   `json:"applicationId"`
	RestaurantID  uint   `json:"restaurantId"`
	Status        string `json:"status"`
	OwnerUserID   uint   `json:"ownerUserId"`
	NewRole       string `json:"newRole"`
}

type RejectResponse struct {
	ApplicationID uint   `json:"applicationId"`
	Status        string `json:"status"`
	Reason        string `json:"reason"`
}

// ====== User สมัครร้าน ======
func (ctl *RestaurantApplicationController) Apply(c *gin.Context) {
	var req ApplyRestaurantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	app := entity.RestaurantApplication{
		Name:                 req.Name,
		Phone:                req.Phone,
		Address:              req.Address,
		Description:          req.Description,
		OpeningTime:          req.OpeningTime,
		ClosingTime:          req.ClosingTime,
		RestaurantCategoryID: req.RestaurantCategoryID,
		OwnerUserID:          utils.CurrentUserID(c),
		Picture:              req.PictureBase64,
		Status:               "pending",
	}

	if err := ctl.DB.Create(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, ApplyResponse{ID: app.ID, Status: "pending"})
}

// ====== Admin ดูรายการ ======
func (ctl *RestaurantApplicationController) List(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")

	var apps []entity.RestaurantApplication
	if err := ctl.DB.
		Preload("OwnerUser").
		Preload("RestaurantCategory").
		Where("status = ?", status).
		Order("id DESC").
		Find(&apps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var resp []RestaurantApplicationResponse
	for _, app := range apps {
		item := RestaurantApplicationResponse{
			ID:          app.ID,
			Name:        app.Name,
			Address:     app.Address,
			Description: app.Description,
			Logo:        app.Picture,
			Phone:       app.Phone,
			OpeningTime: app.OpeningTime,
			ClosingTime: app.ClosingTime,
			SubmittedAt: app.CreatedAt.Format(time.RFC3339),
			Status:      app.Status,
		}
		item.RestaurantCategory.ID = app.RestaurantCategory.ID
		item.RestaurantCategory.Name = app.RestaurantCategory.CategoryName
		item.OwnerUser.FirstName = app.OwnerUser.FirstName
		item.OwnerUser.LastName = app.OwnerUser.LastName
		item.OwnerUser.Email = app.OwnerUser.Email
		item.OwnerUser.PhoneNumber = app.OwnerUser.PhoneNumber

		resp = append(resp, item)
	}

	c.JSON(http.StatusOK, gin.H{"items": resp})
}

// ====== Admin อนุมัติ ======
type ApproveReq struct {
	RestaurantStatusID uint  `json:"restaurantStatusId"`
	AdminID            *uint `json:"adminId,omitempty"`
}

// ====== Admin อนุมัติ ======
func (ctl *RestaurantApplicationController) Approve(c *gin.Context) {
	appID, _ := strconv.Atoi(c.Param("id"))

	// เอา userId ของคนที่ login
	uidAny, ok := c.Get("userId")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := uidAny.(uint)

	// หา admin จาก user_id
	var admin entity.Admin
	if err := ctl.DB.Where("user_id = ?", userID).First(&admin).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "not an admin"})
		return
	}

	// หาใบสมัคร
	var app entity.RestaurantApplication
	if err := ctl.DB.First(&app, uint(appID)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if app.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "application is not pending"})
		return
	}

	// set ค่า status และ admin
	statusID := uint(1)
	now := time.Now()

	rest := entity.Restaurant{
		Name:                 app.Name,
		Address:              app.Address,
		Description:          app.Description,
		Picture:              app.Picture,
		OpeningTime:          app.OpeningTime,
		ClosingTime:          app.ClosingTime,
		RestaurantCategoryID: app.RestaurantCategoryID,
		RestaurantStatusID:   statusID,
		UserID:               app.OwnerUserID,
		AdminID:              &admin.ID, // ✅ ผูกกับ admin จริง ๆ
	}

	tx := ctl.DB.Begin()

	// สร้างร้าน
	if err := tx.Create(&rest).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// update role owner
	if err := tx.Model(&entity.User{}).
		Where("id = ?", app.OwnerUserID).
		Where("role = '' OR role = 'customer'").
		Update("role", "owner").Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// update application
	app.Status = "approved"
	app.ReviewedAt = &now
	app.AdminID = &admin.ID // ✅ ติด admin ที่อนุมัติ
	if err := tx.Save(&app).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	tx.Commit()

	var owner entity.User
	ctl.DB.First(&owner, app.OwnerUserID)

	c.JSON(http.StatusOK, ApproveResponse{
		ApplicationID: uint(appID),
		RestaurantID:  rest.ID,
		Status:        "approved",
		OwnerUserID:   owner.ID,
		NewRole:       owner.Role,
	})
}


// ====== Admin ปฏิเสธ ======
type RejectReq struct {
	Reason  string `json:"reason" binding:"required"`
	AdminID *uint  `json:"adminId,omitempty"`
}

func (ctl *RestaurantApplicationController) Reject(c *gin.Context) {
	appID, _ := strconv.Atoi(c.Param("id"))

	var req RejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var app entity.RestaurantApplication
	if err := ctl.DB.First(&app, appID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if app.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot reject with status " + app.Status})
		return
	}

	now := time.Now()
	app.Status = "rejected"
	app.ReviewedAt = &now
	app.AdminID = req.AdminID
	app.RejectReason = &req.Reason

	if err := ctl.DB.Save(&app).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, RejectResponse{
		ApplicationID: uint(appID),
		Status:        "rejected",
		Reason:        req.Reason,
	})
}
