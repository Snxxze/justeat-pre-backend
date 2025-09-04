// controllers/payment_controller.go
package controllers

import (
	"log"
	"net/http"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"backend/entity"
)

type PaymentController struct{ DB *gorm.DB }
// ... NewPaymentController, uploadSlipReq เหมือนเดิม

func (ctl *PaymentController) UploadSlip(c *gin.Context) {
	var req uploadSlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	if len(req.SlipBase64) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty slip"})
		return
	}
	if len(req.SlipBase64) > 7*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slip too large"})
		return
	}

	// 1) ตรวจว่า order มีจริง
	var order entity.Order
	if err := ctl.DB.First(&order, req.OrderID).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order not found"})
		return
	}

	// 2) โหลด/สร้าง payment
	var p entity.Payment
	if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
		p = entity.Payment{ OrderID: order.ID }
	}

	// 3) อัปเดต
	p.Amount          = req.Amount
	p.SlipBase64      = req.SlipBase64
	p.SlipContentType = req.ContentType // ถ้ามีคอลัมน์นี้ใน DB แล้ว

	// (ทางเลือก) ตั้งสถานะเริ่มต้น หากมีตาราง payment_statuses และมีค่า PENDING
	// var st entity.PaymentStatus
	// if err := ctl.DB.Where("code = ?", "PENDING").First(&st).Error; err == nil {
	//     p.PaymentStatusID = st.ID
	// }

	if err := ctl.DB.Save(&p).Error; err != nil {
		log.Printf("UploadSlip save error: %v", err)
		// ส่งข้อความ error จริงกลับไปชั่วคราวเพื่อดีบัก (อย่าลืมปิดในโปรดักชัน)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "paymentId": p.ID})
}
