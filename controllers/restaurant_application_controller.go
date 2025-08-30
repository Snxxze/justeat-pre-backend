package controllers

import (
	"errors"
	"time"

	"backend/entity"
	"backend/pkg/resp"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RestaurantApplicationController struct{ DB *gorm.DB }

func NewRestaurantApplicationController(db *gorm.DB) *RestaurantApplicationController {
	return &RestaurantApplicationController{DB: db}
}

// ===== ผู้ใช้ยื่นสมัคร =====
type ApplyRestaurantReq struct {
	Name                 string `json:"name" binding:"required"`
	Address              string `json:"address"`
	Description          string `json:"description"`
	Picture              string `json:"picture"`
	RestaurantCategoryID uint   `json:"restaurantCategoryId" binding:"required"`
}

func (ctl *RestaurantApplicationController) Apply(c *gin.Context) {
	var req ApplyRestaurantReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}

	app := entity.RestaurantApplication{
		Name:                 req.Name,
		Address:              req.Address,
		Description:          req.Description,
		Picture:              req.Picture,
		RestaurantCategoryID: req.RestaurantCategoryID,
		OwnerUserID:          utils.CurrentUserID(c),
		Status:               "pending",
	}
	if err := ctl.DB.Create(&app).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.Created(c, gin.H{"id": app.ID, "status": app.Status})
}

// ===== แอดมินดูรายการ/อนุมัติ/ปฏิเสธ =====

func (ctl *RestaurantApplicationController) List(c *gin.Context) {
	status := c.DefaultQuery("status", "pending")
	type row struct {
		ID                   uint      `json:"id"`
		Name                 string    `json:"name"`
		OwnerUserID          uint      `json:"ownerUserId"`
		RestaurantCategoryID uint      `json:"restaurantCategoryId"`
		Status               string    `json:"status"`
		CreatedAt            time.Time `json:"createdAt"`
	}
	var items []row
	if err := ctl.DB.Model(&entity.RestaurantApplication{}).
		Select("id, name, owner_user_id, restaurant_category_id, status, created_at").
		Where("status = ?", status).
		Order("id DESC").Find(&items).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"items": items})
}

type ApproveReq struct {
	RestaurantStatusID uint  `json:"restaurantStatusId"` // ถ้าไม่ส่ง ใช้ 1 (Open)
	AdminID            *uint `json:"adminId,omitempty"`
}

func (ctl *RestaurantApplicationController) Approve(c *gin.Context) {
	id := c.Param("id")

	var app entity.RestaurantApplication
	if err := ctl.DB.First(&app, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.BadRequest(c, "application not found")
			return
		}
		resp.ServerError(c, err)
		return
	}
	if app.Status != "pending" {
		resp.BadRequest(c, "application is not pending")
		return
	}

	var req ApproveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}
	statusID := req.RestaurantStatusID
	if statusID == 0 {
		statusID = 1
	} // e.g. Open

	now := time.Now()
	tx := ctl.DB.Begin()

	// 1) สร้าง Restaurant จริง
	r := entity.Restaurant{
		Name: app.Name, Address: app.Address, Description: app.Description, Picture: app.Picture,
		RestaurantCategoryID: app.RestaurantCategoryID,
		RestaurantStatusID:   statusID,
		UserID:               app.OwnerUserID,
	}
	if req.AdminID != nil {
		r.AdminID = req.AdminID
	}
	if err := tx.Create(&r).Error; err != nil {
		tx.Rollback()
		resp.ServerError(c, err)
		return
	}

	// 2) อัปเกรดสิทธิ์เจ้าของ: customer -> owner (แต่ไม่แตะ admin/owner เดิม)
	var owner entity.User
	if err := tx.First(&owner, app.OwnerUserID).Error; err != nil {
		tx.Rollback()
		resp.ServerError(c, err)
		return
	}
	if owner.Role == "" || owner.Role == "customer" {
		if err := tx.Model(&owner).Update("role", "owner").Error; err != nil {
			tx.Rollback()
			resp.ServerError(c, err)
			return
		}
	}

	// 3) อัปเดตใบสมัคร -> approved
	app.Status = "approved"
	app.ReviewedAt = &now
	app.AdminID = req.AdminID
	if err := tx.Save(&app).Error; err != nil {
		tx.Rollback()
		resp.ServerError(c, err)
		return
	}

	if err := tx.Commit().Error; err != nil {
		resp.ServerError(c, err)
		return
	}

	resp.OK(c, gin.H{
		"applicationId": app.ID,
		"restaurantId":  r.ID,
		"status":        app.Status,
		"ownerUserId":   owner.ID,
		"newRole":       owner.Role, // "owner" ถ้าเพิ่งอัปเกรด
	})
}

type RejectReq struct {
	Reason  string `json:"reason" binding:"required"`
	AdminID *uint  `json:"adminId,omitempty"`
}

func (ctl *RestaurantApplicationController) Reject(c *gin.Context) {
	id := c.Param("id")
	var app entity.RestaurantApplication
	if err := ctl.DB.First(&app, id).Error; err != nil {
		resp.BadRequest(c, "application not found")
		return
	}
	if app.Status != "pending" {
		resp.BadRequest(c, "application is not pending")
		return
	}

	var req RejectReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.BadRequest(c, err.Error())
		return
	}

	now := time.Now()
	app.Status = "rejected"
	app.ReviewedAt = &now
	app.AdminID = req.AdminID
	app.RejectReason = &req.Reason

	if err := ctl.DB.Save(&app).Error; err != nil {
		resp.ServerError(c, err)
		return
	}
	resp.OK(c, gin.H{"applicationId": app.ID, "status": app.Status})
}
