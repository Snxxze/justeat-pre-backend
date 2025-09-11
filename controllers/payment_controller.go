package controllers

import (
	"fmt"
	"log" // เพิ่มการ import log
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"backend/entity"
)

type PaymentController struct {
	DB *gorm.DB
}

func NewPaymentController(db *gorm.DB) *PaymentController {
	return &PaymentController{DB: db}
}

type uploadSlipReq struct {
	OrderID     uint   `json:"orderId"`
	Amount      int64  `json:"amount"`
	ContentType string `json:"contentType"`
	SlipBase64  string `json:"slipBase64"`
}

func (ctl *PaymentController) UploadSlip(c *gin.Context) {
	log.Println("📸 UploadSlip endpoint called") // Debug log
	
	var req uploadSlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ JSON bind error: %v", err) // Debug log
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}
	
	log.Printf("📝 Request data: OrderID=%d, Amount=%d, ContentType=%s, Base64 length=%d", 
		req.OrderID, req.Amount, req.ContentType, len(req.SlipBase64)) // Debug log
		
	if len(req.SlipBase64) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty slip"})
		return
	}
	// จำกัด ~5MB ไฟล์จริง (base64 ~ +33% ≈ 6.7MB)
	if len(req.SlipBase64) > 7*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "slip too large"})
		return
	}

	// 1) ตรวจว่า order มีจริง
	var order entity.Order
	if err := ctl.DB.First(&order, req.OrderID).Error; err != nil {
		log.Printf("❌ Order not found: %v", err) // Debug log
		c.JSON(http.StatusBadRequest, gin.H{"error": "order not found"})
		return
	}

	log.Printf("✅ Order found: %+v", order) // Debug log

	// 2) โหลด/สร้าง payment ผูกกับออเดอร์นี้
	var p entity.Payment
	if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
		log.Println("📄 Creating new payment") // Debug log
		p = entity.Payment{OrderID: order.ID}
	} else {
		log.Printf("📄 Found existing payment: %+v", p) // Debug log
	}

	// 3) อัปเดตค่าและบันทึก
	p.Amount = req.Amount
	p.SlipBase64 = req.SlipBase64
	p.SlipContentType = req.ContentType

	if err := ctl.DB.Save(&p).Error; err != nil {
		log.Printf("❌ Save error: %v", err) // Debug log
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save failed"})
		return
	}

	log.Printf("✅ Payment saved successfully: ID=%d", p.ID) // Debug log
	c.JSON(http.StatusOK, gin.H{
    "success": true,
    "slipData": gin.H{
        "paymentId": p.ID,
        "amount": float64(req.Amount) / 100, // แปลงจากสตางค์กลับเป็นบาท
        "transRef": fmt.Sprintf("TXN-%d", p.ID),
    },
})
}