package controllers

import (
    "encoding/base64"
    "fmt"
    "log"
    "net/http"
	"strings"

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

// 🔥 ปรับ struct ให้ตรงกับ Frontend
type uploadSlipReq struct {
    OrderID     int    `json:"orderId" binding:"required"`     // เปลี่ยนเป็น int
    Amount      int    `json:"amount" binding:"required,min=1"` // เปลี่ยนเป็น int และเพิ่ม validation
    ContentType string `json:"contentType" binding:"required"`
    SlipBase64  string `json:"slipBase64" binding:"required"`
}

// 🔥 เพิ่ม Response struct ให้ชัดเจน
type uploadSlipResponse struct {
    Success  bool `json:"success"`
    SlipData *struct {
        PaymentID int     `json:"paymentId"`
        Amount    float64 `json:"amount"`
        TransRef  string  `json:"transRef"`
    } `json:"slipData,omitempty"`
    Error            string   `json:"error,omitempty"`
    ValidationErrors []string `json:"validationErrors,omitempty"`
}

func (ctl *PaymentController) UploadSlip(c *gin.Context) {
    log.Println("📸 UploadSlip endpoint called")
    
    var req uploadSlipReq
    if err := c.ShouldBindJSON(&req); err != nil {
        log.Printf("JSON bind error: %v", err)
        c.JSON(http.StatusBadRequest, uploadSlipResponse{
            Success: false,
            Error:   "Invalid request format: " + err.Error(),
        })
        return
    }
    
    log.Printf("📝 Request data: OrderID=%d, Amount=%d, ContentType=%s, Base64 length=%d", 
        req.OrderID, req.Amount, req.ContentType, len(req.SlipBase64))
    
    // 🔥 เพิ่มการตรวจสอบ base64 ว่าเป็น valid image
    imageData, err := base64.StdEncoding.DecodeString(req.SlipBase64)
    if err != nil {
        log.Printf("Invalid base64: %v", err)
        c.JSON(http.StatusBadRequest, uploadSlipResponse{
            Success: false,
            Error:   "Invalid base64 image data",
        })
        return
    }
    
    // ตรวจสอบขนาดไฟล์จริง (5MB)
    if len(imageData) > 5*1024*1024 {
        c.JSON(http.StatusBadRequest, uploadSlipResponse{
            Success: false,
            Error:   "Image size exceeds 5MB limit",
        })
        return
    }
    
    // 🔥 เพิ่มการตรวจสอบ content type
    if req.ContentType == "" || (!strings.HasPrefix(req.ContentType, "image/")) {
        c.JSON(http.StatusBadRequest, uploadSlipResponse{
            Success: false,
            Error:   "Invalid content type, must be image/*",
        })
        return
    }

    // 1) ตรวจว่า order มีจริง
    var order entity.Order
    if err := ctl.DB.First(&order, req.OrderID).Error; err != nil {
        log.Printf("Order not found: %v", err)
        if err == gorm.ErrRecordNotFound {
            c.JSON(http.StatusNotFound, uploadSlipResponse{
                Success: false,
                Error:   fmt.Sprintf("Order ID %d not found", req.OrderID),
            })
        } else {
            c.JSON(http.StatusInternalServerError, uploadSlipResponse{
                Success: false,
                Error:   "Database error while finding order",
            })
        }
        return
    }

    log.Printf("Order found: ID=%d, Status=%s", order.ID, order.OrderStatusID)

    // 2) โหลด/สร้าง payment ผูกกับออเดอร์นี้
    var p entity.Payment
    if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            log.Println("Creating new payment")
            p = entity.Payment{OrderID: order.ID}
        } else {
            log.Printf("Database error: %v", err)
            c.JSON(http.StatusInternalServerError, uploadSlipResponse{
                Success: false,
                Error:   "Database error while finding payment",
            })
            return
        }
    } else {
        log.Printf("Found existing payment: ID=%d", p.ID)
    }

    // 3) อัปเดตค่าและบันทึก
    p.Amount = int64(req.Amount) // แปลง int เป็น int64
    p.SlipBase64 = req.SlipBase64
    p.SlipContentType = req.ContentType

    if err := ctl.DB.Save(&p).Error; err != nil {
        log.Printf("Save error: %v", err)
        c.JSON(http.StatusInternalServerError, uploadSlipResponse{
            Success: false,
            Error:   "Failed to save payment data",
        })
        return
    }

    log.Printf("Payment saved successfully: ID=%d", p.ID)
    
    // 🔥 ปรับ Response ให้ตรงกับ Frontend
    response := uploadSlipResponse{
        Success: true,
        SlipData: &struct {
            PaymentID int     `json:"paymentId"`
            Amount    float64 `json:"amount"`
            TransRef  string  `json:"transRef"`
        }{
            PaymentID: int(p.ID),
            Amount:    float64(req.Amount) / 100, // แปลงจากสตางค์กลับเป็นบาท
            TransRef:  fmt.Sprintf("TXN-%d", p.ID),
        },
    }
    
    c.JSON(http.StatusOK, response)
}
