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
	"math"
	"time"
)

var paidStatus entity.PaymentStatus

type PaymentController struct {
	DB            *gorm.DB
	EasySlipToken string
	httpClient    *http.Client

	paidStatusID uint
}

// ===== Helper สำหรับตัด header data:image/...;base64, =====
func stripDataURLHeader(b64 string) (string, error) {
	if i := strings.Index(b64, ","); i != -1 {
		cleaned := b64[i+1:]
		// ตรวจสอบว่าเป็น base64 ที่ถูกต้อง
		if _, err := base64.StdEncoding.DecodeString(cleaned); err != nil {
			return "", fmt.Errorf("invalid base64 format")
		}
		return cleaned, nil
	}

	// ตรวจสอบ base64 โดยตรง
	if _, err := base64.StdEncoding.DecodeString(b64); err != nil {
		return "", fmt.Errorf("invalid base64 format")
	}
	return b64, nil
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
	OrderID     uint   `json:"orderId" binding:"required"`      // เปลี่ยนเป็น int
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
	var req uploadSlipReq
	var order entity.Order
	var p entity.Payment

	log.Println(" UploadSlip endpoint called")

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("JSON bind error: %v", err)
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Invalid request format: " + err.Error(),
		})
		return
	}

	log.Printf(" Request data: OrderID = %d, Amount = %d, ContentType = %s, Base64 length = %d",
		req.OrderID, req.Amount, req.ContentType, len(req.SlipBase64))

	// --- ตัด header + ตรวจ base64 ---
	cleanB64, err := stripDataURLHeader(req.SlipBase64)
	if err != nil {
		log.Printf("Invalid base64: %v", err)
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Invalid base64 image data",
		})
		return
	}

	// decode เพื่อตรวจขนาดไฟล์ (5MB)
	imageData, err := base64.StdEncoding.DecodeString(cleanB64)
	if err != nil {
		log.Printf("Invalid base64 after strip: %v", err)
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Invalid base64 image data",
		})
		return
	}
	if len(imageData) > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Image size exceeds 5MB limit",
		})
		return
	}

	// ตรวจ content type
	if req.ContentType == "" || (!strings.HasPrefix(req.ContentType, "image/")) {
		c.JSON(http.StatusBadRequest, uploadSlipResponse{
			Success: false,
			Error:   "Invalid content type, must be image/*",
		})
		return
	}

	// 1) ตรวจว่า order มีจริง
	if err := ctl.DB.First(&order, uint(req.OrderID)).Error; err != nil {
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

	// 2) โหลด/สร้าง payment ผูกกับออเดอร์นี้
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

	// 2.1) ถ้าชำระสำเร็จแล้ว ไม่ให้ส่งอีก
	if p.PaidAt != nil {
		c.JSON(http.StatusConflict, uploadSlipResponse{
			Success: false,
			Error:   "already_paid",
		})
		return
	}

	// 3) อัปเดตค่าและบันทึก (ใช้ base64 ที่ strip แล้วเสมอ)
	p.Amount = int64(req.Amount) // แนะนำให้ทำเป็น int64 หน่วยสตางค์ทั้งระบบ
	p.SlipBase64 = cleanB64      // <— ใช้ cleanB64 แทน req.SlipBase64
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

	// Response กลับไปยัง frontend
	response := uploadSlipResponse{
		Success: true,
		SlipData: &struct {
			PaymentID int     `json:"paymentId"`
			Amount    float64 `json:"amount"`
			TransRef  string  `json:"transRef"`
		}{
			PaymentID: int(p.ID),
			Amount:    float64(req.Amount) / 100.0, // สตางค์ -> บาท
			TransRef:  fmt.Sprintf("TXN-%d", p.ID), // ยังไม่มีจาก EasySlip ใน endpoint นี้
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
	Amount float64 `json:"amount"` // บาท
	Local  struct {
		Amount   float64 `json:"amount"` // บาท
		Currency string  `json:"currency"`
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
	Amount         int64  `json:"amount"`
	ContentType    string `json:"contentType"` // image/png, image/jpeg
	SlipBase64     string `json:"slipBase64" binding:"required"`
	CheckDuplicate *bool  `json:"checkDuplicate,omitempty"`
}

// POST /api/payments/verify-easyslip
func (ctl *PaymentController) VerifyEasySlip(c *gin.Context) {
	// 1) bind + validate
	var req verifySlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request"})
		return
	}
	if req.SlipBase64 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_base64"})
		return
	}
	if req.CheckDuplicate == nil {
		def := true
		req.CheckDuplicate = &def
	}
	if ctl.EasySlipToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "missing_easyslip_token"})
		return
	}

	// 2) โหลด order + payment ก่อน แล้วเช็ค already_paid “ตรงนี้”
	var order entity.Order
	if err := ctl.DB.First(&order, uint(req.OrderID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("order %d not found", req.OrderID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_find_order_error"})
		}
		return
	}

	var p entity.Payment
	if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			p = entity.Payment{OrderID: order.ID}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_find_payment_error"})
			return
		}
	}
	log.Printf("[VERIFY] order = %d paymentID = %d already_paid = %v", order.ID, p.ID, p.PaidAt != nil)

	// ✅ ถ้าจ่ายแล้ว ตัดจบทันที — อย่าเรียก EasySlip
	if p.PaidAt != nil {
		log.Printf("[UPLOAD] Order %d already paid at %v", req.OrderID, *p.PaidAt)
		c.JSON(http.StatusConflict, gin.H{
			"success":   false,
			"error":     "already_paid",
			"paymentId": p.ID,
			"transRef":  p.TransRef, // ← เพิ่มให้ FE เอาไปโชว์ได้
			"paidAt":    p.PaidAt,   // ← เพิ่มให้ FE เอาไปโชว์ได้
		})
		return
	}

	// 3) เตรียมรูปแล้วค่อยเรียก EasySlip
	b64, err := stripDataURLHeader(req.SlipBase64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_base64"})
		return
	}

	body, _ := json.Marshal(easySlipVerifyReq{
		Image:          b64,
		CheckDuplicate: req.CheckDuplicate,
	})

	ctx, cancel := context.WithTimeout(c.Request.Context(), 25*time.Second)
	defer cancel()

	httpReq, _ := http.NewRequestWithContext(ctx, "POST",
		"https://developer.easyslip.com/api/v1/verify", bytes.NewReader(body))
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

		rawBaht := ok.Data.Amount.Amount
		slipSatang := int64(math.Round(rawBaht * 100))
		matched := (req.Amount == 0) || (slipSatang == req.Amount)

		// ---- ถ้ายอดไม่ตรง: ไม่แตะ DB, ตอบ 400 amount_mismatch ----
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{
				"success":        false,
				"error":          "amount_mismatch",
				"expectedSatang": req.Amount,
				"expectedBaht":   float64(req.Amount) / 100.0,
				"slipData": gin.H{
					"amountSatang": slipSatang,
					"amountBaht":   rawBaht,
					"date":         ok.Data.Date,
					"transRef":     ok.Data.TransRef,
				},
			})
			return
		}

		// ---- ตรงกัน: ค่อยบันทึกลง DB ----
		if req.ContentType == "" || !strings.HasPrefix(req.ContentType, "image/") {
			req.ContentType = "image/*"
		}

		p.SlipBase64 = b64
		p.SlipContentType = req.ContentType
		p.Amount = slipSatang
		p.TransRef = ok.Data.TransRef

		now := time.Now()
		p.PaidAt = &now
		if err := ctl.DB.Where("status_name = ?", "Paid").First(&paidStatus).Error; err == nil {
			p.PaymentStatusID = paidStatus.ID
		}

		if err := ctl.DB.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_save_payment_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"matchedAmount":  true,
			"paymentId":      p.ID,
			"expectedSatang": req.Amount,
			"expectedBaht":   float64(req.Amount) / 100.0,
			"slipData": gin.H{
				"amountSatang": slipSatang,
				"amountBaht":   rawBaht,
				"date":         ok.Data.Date,
				"transRef":     ok.Data.TransRef,
				"sender":       ok.Data.Sender,
				"receiver":     ok.Data.Receiver,
				"payload":      ok.Data.Payload,
			},
		})
		return
	}

	var er easySlipErrResp
	if err := json.NewDecoder(resp.Body).Decode(&er); err != nil {
		c.JSON(resp.StatusCode, gin.H{"error": "easyslip_error"})
		return
	}
	if er.Message == "duplicate_slip" && er.Data != nil {
		// ---- สลิปซ้ำ: ตัดสินจาก "ยอด" ว่าจะยอมรับเลย หรือฟ้อง mismatch ----
		dupBaht := er.Data.Amount.Amount
		dupSatang := int64(math.Round(dupBaht * 100))
		matched := (req.Amount == 0) || (dupSatang == req.Amount)

		if !matched {
			// ❌ duplicate แต่ยอด "ไม่ตรง" → ฟ้อง amount_mismatch (ไม่แตะ DB)
			c.JSON(http.StatusBadRequest, gin.H{
				"success":        false,
				"error":          "amount_mismatch",
				"expectedSatang": req.Amount,
				"expectedBaht":   float64(req.Amount) / 100.0,
				"slipData": gin.H{
					"amountSatang": dupSatang,
					"amountBaht":   dupBaht,
					"date":         er.Data.Date,
					"transRef":     er.Data.TransRef,
				},
			})
			return
		}

		// ✅ duplicate แต่ยอด "ตรง" → treat as success (idempotent) แล้วบันทึกลง DB
		if req.ContentType == "" || !strings.HasPrefix(req.ContentType, "image/") {
			req.ContentType = "image/*"
		}
		p.SlipBase64 = b64 // เก็บรูปไว้เพื่อ audit
		p.SlipContentType = req.ContentType
		p.Amount = dupSatang
		p.TransRef = er.Data.TransRef // แนะนำทำ unique index ที่ DB (ไว้ค่อยเพิ่ม)

		now := time.Now()
		p.PaidAt = &now
		if err := ctl.DB.Where("status_name = ?", "Paid").First(&paidStatus).Error; err == nil {
			p.PaymentStatusID = paidStatus.ID
		}

		if err := ctl.DB.Save(&p).Error; err != nil {
			// ถ้าเจอ unique-constraint ของ transRef ก็กันการผูกซ้ำโดยธรรมชาติ
			c.JSON(http.StatusInternalServerError, gin.H{"error": "db_save_payment_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"matchedAmount":  true,
			"paymentId":      p.ID,
			"expectedSatang": req.Amount,
			"expectedBaht":   float64(req.Amount) / 100.0,
			"slipData": gin.H{
				"amountSatang": dupSatang,
				"amountBaht":   dupBaht,
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
