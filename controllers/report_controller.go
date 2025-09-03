package controllers

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"backend/entity"
	"backend/services"

	"github.com/gin-gonic/gin"
)

type ReportController struct {
    reportService *services.ReportService
}

func NewReportController(service *services.ReportService) *ReportController {
    return &ReportController{reportService: service}
}

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

    report, err := rc.reportService.CreateReport(
        userID,
        req.Name,
        req.Email,
        req.PhoneNumber,
        req.Description,
        picturePath,
        req.IssueTypeID,
    )
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot save report"})
        return
    }

    c.JSON(http.StatusCreated, gin.H{"ok": true, "report": report})
}

// GET /reports (เฉพาะ user คนนั้น)
func (rc *ReportController) ListReports(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	var reports []entity.Report
	if err := rc.reportService.FindAllByUser(userID, &reports); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "cannot fetch reports"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "reports": reports})
}

// GET /reports/:id
func (rc *ReportController) GetReportByID(c *gin.Context) {
	userIDAny, _ := c.Get("userId")
	userID := userIDAny.(uint)

	id := c.Param("id")

	report, err := rc.reportService.FindByIDAndUser(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "report not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "report": report})
}
