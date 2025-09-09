package controllers

import (
	"errors"
	"strconv"
	"time"

	"backend/entity"
	"backend/pkg/resp"
	"backend/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RiderController struct{ DB *gorm.DB }

func NewRiderController(db *gorm.DB) *RiderController { return &RiderController{DB: db} }

func (rc *RiderController) riderByUser(c *gin.Context) (*entity.Rider, bool) {
	uid := utils.CurrentUserID(c)
	var r entity.Rider
	if err := rc.DB.Select("id, user_id, vehicle_plate, rider_status_id").
		Where("user_id = ?", uid).First(&r).Error; err != nil {
	if errors.Is(err, gorm.ErrRecordNotFound) {
			resp.BadRequest(c, "rider profile not found")
			return nil, false
		}
		resp.ServerError(c, err)
		return nil, false
	}
	return &r, true
}

func (rc *RiderController) Dashboard(c *gin.Context) {
	r, ok := rc.riderByUser(c); if !ok { return }
	var active int64
	rc.DB.Model(&entity.RiderWork{}).
		Where("rider_id = ? AND finish_at IS NULL", r.ID).
		Count(&active)
	resp.OK(c, gin.H{
		"activeJobs":   active,
		"riderStatusId": r.RiderStatusID,
	})
}

func (rc *RiderController) JobList(c *gin.Context) {
	r, ok := rc.riderByUser(c); if !ok { return }
	type row struct {
		ID           uint       `json:"id"`
		OrderID      uint       `json:"orderId"`
		RestaurantID uint       `json:"restaurantId"`
		WorkAt       *time.Time `json:"workAt"`
	}
	var jobs []row
	if err := rc.DB.Model(&entity.RiderWork{}).
		Select("id, order_id, restaurant_id, work_at").
		Where("rider_id = ? AND finish_at IS NULL", r.ID).
		Order("id DESC").
		Find(&jobs).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": jobs})
}

func (rc *RiderController) Histories(c *gin.Context) {
	r, ok := rc.riderByUser(c); if !ok { return }
	type row struct {
		ID       uint       `json:"id"`
		OrderID  uint       `json:"orderId"`
		FinishAt *time.Time `json:"finishAt"`
	}
	var items []row
	if err := rc.DB.Model(&entity.RiderWork{}).
		Select("id, order_id, finish_at").
		Where("rider_id = ? AND finish_at IS NOT NULL", r.ID).
		Order("id DESC").
		Limit(100).
		Find(&items).Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"items": items})
}

func (rc *RiderController) Profile(c *gin.Context) {
	r, ok := rc.riderByUser(c); if !ok { return }

	// เพิ่มข้อมูลผู้ใช้
	var u entity.User
	if err := rc.DB.Select("id, first_name, last_name, email, phone").
		Where("id = ?", r.UserID).
		First(&u).Error; err != nil {
		resp.ServerError(c, err); return
	}

	resp.OK(c, gin.H{
		"id":            r.ID,
		"vehiclePlate":  r.VehiclePlate,
		"riderStatusId": r.RiderStatusID,
		"user": gin.H{
			"id":        u.ID,
			"firstName": u.FirstName,
			"lastName":  u.LastName,
			"email":     u.Email,
			"phoneNumber":     u.PhoneNumber,
		},
	})
}

func (rc *RiderController) FinishJob(c *gin.Context) {
	r, ok := rc.riderByUser(c); if !ok { return }

	idStr := c.Param("id")
	jobID64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		resp.BadRequest(c, "invalid job id"); return
	}
	jobID := uint(jobID64)

	var w entity.RiderWork
	if err := rc.DB.Where("id = ? AND rider_id = ?", jobID, r.ID).First(&w).Error; err != nil {
		resp.BadRequest(c, "job not found"); return
	}

	now := time.Now()
	tx := rc.DB.Begin()
	if err := tx.Model(&w).Update("finish_at", &now).Error; err != nil {
		tx.Rollback(); resp.ServerError(c, err); return
	}
	// อัปเดตสถานะคำสั่งซื้อเป็น Completed (4) (ปรับตาม seed ของคุณ)
	if err := tx.Model(&entity.Order{}).
		Where("id = ?", w.OrderID).
		Update("order_status_id", 4).Error; err != nil {
		tx.Rollback(); resp.ServerError(c, err); return
	}
	if err := tx.Commit().Error; err != nil {
		resp.ServerError(c, err); return
	}
	resp.OK(c, gin.H{"jobId": w.ID, "finished": true})
}
