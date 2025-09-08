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

	"bytes"
	"context"
	"encoding/json"
	"time"
)

var paidStatus entity.PaymentStatus

type PaymentController struct {
	DB            *gorm.DB
	EasySlipToken string
	httpClient    *http.Client

	paidStatusID uint
}

// ===== Helpers เพิ่ม helper สำหรับตัด header data:image/...;base64, =====
func stripDataURLHeader(b64 string) string {
	// ตัดส่วน "data:image/png;base64," ออก ถ้ามี
	if i := strings.Index(b64, ","); i != -1 {
		return b64[i+1:]
	}
	return b64
}

func NewPaymentController(db *gorm.DB, easySlipToken string) *PaymentController {
	log.Printf("[PAYMENT_CONTROLLER] Received token length: %d", len(easySlipToken))
	if easySlipToken == "" {
		log.Printf("[PAYMENT_CONTROLLER] WARNING: EasySlip token is empty!")
	}
	return &PaymentController{
		DB:            db,
		EasySlipToken: easySlipToken,
		httpClient:    &http.Client{Timeout: 20 * time.Second},
	}
}

// ปรับ struct ให้ตรงกับ Frontend
type uploadSlipReq struct {
	OrderID     int    `json:"orderId" binding:"required"`      // เปลี่ยนเป็น int
	Amount      int    `json:"amount" binding:"required,min=1"` // เปลี่ยนเป็น int และเพิ่ม validation
	ContentType string `json:"contentType" binding:"required"`
	SlipBase64  string `json:"slipBase64" binding:"required"`
}

// เพิ่ม Response struct ให้ชัดเจน
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
	log.Println(" UploadSlip endpoint called")

	var req uploadSlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSON bind error: %v", err)
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
		})
		return
	}

	log.Printf(" Request data: OrderID=%d, Amount=%d, ContentType=%s, Base64 length=%d",
		req.OrderID, req.Amount, req.ContentType, len(req.SlipBase64))

	//  เพิ่มการตรวจสอบ base64 ว่าเป็น valid image
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

	//  เพิ่มการตรวจสอบ content type
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

	// log.Printf("Order found: ID=%d, Status=%s", order.ID, order.OrderStatusID)

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

	//  ปรับ Response ให้ตรงกับ Frontend
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

// ====== โครงสร้างสำหรับเรียก EasySlip (base64) ======

type easySlipVerifyReq struct {
	Image          string `json:"image"`
	CheckDuplicate *bool  `json:"checkDuplicate,omitempty"`
}

type EasySlipAmount struct {
	Amount int64 `json:"amount"`
	Local  struct {
		Amount   int64  `json:"amount"`
		Currency string `json:"currency"`
	} `json:"local"`
}

type EasySlipBank struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Short string `json:"short"`
}

type EasySlipAccount struct {
	Name struct {
		TH string `json:"th"`
		EN string `json:"en"`
	} `json:"name"`
	Bank *struct {
		Type    string `json:"type"`
		Account string `json:"account"`
	} `json:"bank,omitempty"`
	Proxy *struct {
		Type    string `json:"type"`
		Account string `json:"account"`
	} `json:"proxy,omitempty"`
}

type EasySlipData struct {
	Payload     string         `json:"payload"`
	TransRef    string         `json:"transRef"`
	Date        string         `json:"date"`
	CountryCode string         `json:"countryCode"`
	Amount      EasySlipAmount `json:"amount"`
	Fee         int64          `json:"fee"`
	Ref1        string         `json:"ref1"`
	Ref2        string         `json:"ref2"`
	Ref3        string         `json:"ref3"`
	Sender      struct {
		Bank    EasySlipBank    `json:"bank"`
		Account EasySlipAccount `json:"account"`
	} `json:"sender"`
	Receiver struct {
		Bank    EasySlipBank    `json:"bank"`
		Account EasySlipAccount `json:"account"`
	} `json:"receiver"`
}

type easySlipOKResp struct {
	Status int          `json:"status"`
	Data   EasySlipData `json:"data"`
}

type easySlipErrResp struct {
	Status  int           `json:"status"`
	Message string        `json:"message"` // duplicate_slip, invalid_image, unauthorized, quota_exceeded, ...
	Data    *EasySlipData `json:"data,omitempty"`
}

// ====== Request จาก frontend เวลา verify ======
type verifySlipReq struct {
	OrderID        int    `json:"orderId" binding:"required"`
	AmountSatang   int64  `json:"amount"`      // สตางค์ (อาจ 0 = ไม่เช็ค)
	ContentType    string `json:"contentType"` // image/png, image/jpeg
	SlipBase64     string `json:"slipBase64" binding:"required"`
	CheckDuplicate *bool  `json:"checkDuplicate,omitempty"`
}

// POST /api/payments/verify-easyslip
func (ctl *PaymentController) VerifyEasySlip(c *gin.Context) {
	log.Printf("[VERIFY] token len=%d", len(ctl.EasySlipToken))
	if ctl.EasySlipToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing_easyslip_token"})
		return
	}

	var req verifySlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if req.SlipBase64 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_base64"})
		return
	}
	// default checkDuplicate = true (ออปชัน)
	if req.CheckDuplicate == nil {
		def := true
		req.CheckDuplicate = &def
	}

	// --- เตรียม base64 ส่งให้ EasySlip (ต้องไม่มี header) ---
	b64 := stripDataURLHeader(req.SlipBase64)

	// ✅ เรียก EasySlip
	body, _ := json.Marshal(easySlipVerifyReq{
		Image:          b64,
		CheckDuplicate: req.CheckDuplicate,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://developer.easyslip.com/api/v1/verify", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+ctl.EasySlipToken)

	resp, err := ctl.httpClient.Do(httpReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "easyslip_unreachable"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var ok easySlipOKResp
		if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "easyslip_decode_error"})
			return
		}

		// ✅ ตรวจยอด ถ้าฝั่ง frontend ส่งมาเป็นสตางค์
		matched := (req.AmountSatang == 0) || (ok.Data.Amount.Amount == req.AmountSatang)

		// ====== NEW: บันทึก/อัปเดต Payment ลง DB ======
		// 1) เช็คว่า Order มีจริง
		var order entity.Order
		if err := ctl.DB.First(&order, req.OrderID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("order %d not found", req.OrderID)})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db_find_order_error"})
			}
			return
		}

		// 2) หา Payment เดิมของออเดอร์นี้ หรือสร้างใหม่
		var p entity.Payment
		if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				p = entity.Payment{OrderID: order.ID}
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "db_find_payment_error"})
				return
			}
		}

		// 3) อัปเดตฟิลด์
		// - เก็บ base64 ที่ไม่มี header เพื่อลดขนาด
		// - ContentType จาก client (ควรเริ่มด้วย "image/")
		if req.ContentType == "" || !strings.HasPrefix(req.ContentType, "image/") {
			// ไม่ fail แต่อย่างน้อยให้บันทึกเป็น image/* เพื่อความปลอดภัย
			req.ContentType = "image/*"
		}
		p.SlipBase64 = b64
		p.SlipContentType = req.ContentType

		// - ยอดใช้ของจริงจากสลิป (สตางค์) เพื่อกัน client ปลอมยอด
		p.Amount = ok.Data.Amount.Amount

		// - ถ้ายอดตรง (หรือไม่ได้บังคับตรวจ) ให้ถือว่าชำระสำเร็จ -> set PaidAt
		if matched {
			now := time.Now()
			p.PaidAt = &now
            log.Printf("[VERIFY] set PaidAt=%s", now.Format(time.RFC3339))

			// (ออปชัน) ตั้งสถานะเป็น PAID ถ้าคุณมีตาราง lookup และ seed ไว้
			if err := ctl.DB.Where("status_name = ?", "Paid").First(&paidStatus).Error; err == nil {
				p.PaymentStatusID = paidStatus.ID
			} else {
				log.Printf("[VERIFY] PAID status not found: %v", err)
			}
		}

		if err := ctl.DB.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_save_payment_error"})
			return
		}

		// 4) ตอบกลับ (แนบ paymentId ให้ frontend ใช้)
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"paymentId":      p.ID,
			"matchedAmount":  matched,
			"expectedSatang": req.AmountSatang,
			"slipData": gin.H{
				"amountSatang": ok.Data.Amount.Amount, // จากสลิป
				"date":         ok.Data.Date,
				"transRef":     ok.Data.TransRef,
				"sender":       ok.Data.Sender,
				"receiver":     ok.Data.Receiver,
				"payload":      ok.Data.Payload,
			},
		})
		return
	}

	// ❌ non-200 : อ่าน error รายละเอียด
	var er easySlipErrResp
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		c.JSON(resp.StatusCode, gin.H{"error": "easyslip_error"})
		return
	}

	if er.Message == "duplicate_slip" && er.Data != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "duplicate_slip",
			"slipData": gin.H{
				"amountSatang": er.Data.Amount.Amount,
				"date":         er.Data.Date,
				"transRef":     er.Data.TransRef,
			},
		})
		return
	}

	c.JSON(resp.StatusCode, gin.H{
		"success": false,
		"error":   er.Message, // invalid_image / unauthorized / quota_exceeded / ...
	})
}
