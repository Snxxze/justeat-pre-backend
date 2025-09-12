// controllers/report_controller.go
package controllers

import (
	"backend/entity"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ReportController struct {
	DB *gorm.DB
}

func NewReportController(db *gorm.DB) *ReportController {
	return &ReportController{DB: db}
}

// ---------- Create ----------
func (rc *ReportController) CreateReport(c *gin.Context) {
	userIDAny, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userID := userIDAny.(uint)

	var req struct {
		Name        string `form:"name"`
		Email       string `form:"email"`
		PhoneNumber string `form:"phoneNumber"`
		Description string `form:"description"`
		IssueTypeID uint   `form:"issueTypeId"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ✅ อัปโหลดไฟล์รูป (optional)
	picturePath := ""
	file, err := c.FormFile("pictures")
	if err == nil {
		filename := fmt.Sprintf("report_%d_%d%s", userID, time.Now().UnixNano(), filepath.Ext(file.Filename))
		savePath := filepath.Join("uploads", "reports", filename)
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save file"})
			return
		}
		picturePath = savePath
	}

	now := time.Now()
	report := &entity.Report{
		Name:        req.Name,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Description: req.Description,
		Picture:     picturePath,
		IssueTypeID: req.IssueTypeID,
		UserID:      userID,
		DateAt:      &now,
		Status:      "pending",
	}

	if err := rc.DB.Create(report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save report"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"ok": true, "report": report})
}

// ---------- User: GET /reports ----------
func (rc *ReportController) ListReports(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	var reports []entity.Report
	if err := rc.DB.Where("user_id = ?", userID).Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot fetch reports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "reports": reports})
}

// ---------- User: GET /reports/:id ----------
func (rc *ReportController) GetReportByID(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	id := c.Param("id")
	var report entity.Report
	if err := rc.DB.Where("id = ? AND user_id = ?", id, userID).First(&report).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "report": report})
}

// ---------- Admin: GET /admin/reports ----------
func (rc *ReportController) ListAllReports(c *gin.Context) {
	var reports []entity.Report
	if err := rc.DB.Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot fetch all reports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "reports": reports})
}

// ---------- Admin: PATCH /admin/reports/:id/status ----------
func (rc *ReportController) UpdateReportStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid report id"})
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// ✅ validate status
	validStatuses := map[string]bool{
		"pending":      true,
		"in_progress":  true,
		"resolved":     true,
		"closed":       true,
	}
	if !validStatuses[req.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}

	if err := rc.DB.Model(&entity.Report{}).
		Where("id = ?", id).
		Update("status", req.Status).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "status updated"})
}
