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
	"errors"
	"math"
	"strconv"
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

// เก็บเฉพาะตัวเลข 0-9 (ใช้กับ PromptPay ที่อาจมี dash/space ติดมา)
func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
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
	OrderID     uint   `json:"orderId" binding:"required"`
	Amount      int    `json:"amount" binding:"required,min=1"`
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
	var pmt entity.Payment
	if err := ctl.DB.Where("order_id = ?", order.ID).First(&pmt).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Println("Creating new payment")
			pmt = entity.Payment{OrderID: order.ID}
		} else {
			log.Printf("Database error: %v", err)
			c.JSON(http.StatusInternalServerError, uploadSlipResponse{
				Success: false,
				Error:   "Database error while finding payment",
			})
			return
		}
	} else {
		log.Printf("Found existing payment: ID=%d", pmt.ID)
	}

	// 2.1) ถ้าชำระสำเร็จแล้ว ไม่ให้ส่งอีก
	if pmt.PaidAt != nil {
		c.JSON(http.StatusConflict, uploadSlipResponse{
			Success: false,
			Error:   "already_paid",
		})
		return
	}

	// 3) อัปเดตค่าและบันทึก (ใช้ base64 ที่ strip แล้วเสมอ)
	pmt.Amount = int64(req.Amount) // เก็บเป็น "บาทจำนวนเต็ม" ให้สอดคล้อง VerifyEasySlip
	pmt.SlipBase64 = cleanB64
	pmt.SlipContentType = req.ContentType

	if err := ctl.DB.Save(&pmt).Error; err != nil {
		log.Printf("Save error: %v", err)
		c.JSON(http.StatusInternalServerError, uploadSlipResponse{
			Success: false,
			Error:   "Failed to save payment data",
		})
		return
	}

	log.Printf("Payment saved successfully: ID=%d", pmt.ID)

	// Response กลับไปยัง frontend
	response := uploadSlipResponse{
		Success: true,
		SlipData: &struct {
			PaymentID int     `json:"paymentId"`
			Amount    float64 `json:"amount"`
			TransRef  string  `json:"transRef"`
		}{
			PaymentID: int(pmt.ID),
			Amount:    float64(req.Amount),
			TransRef:  fmt.Sprintf("TXN-%d", pmt.ID), // ยังไม่มีจาก EasySlip ใน endpoint นี้
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
	Message string        `json:"message"` // duplicate_slip, invalid_image, qrcode_not_found, unauthorized, quota_exceeded, ...
	Data    *EasySlipData `json:"data,omitempty"`
}

// ====== Request จาก frontend เวลา verify ======
type verifySlipReq struct {
	OrderID        int    `json:"orderId" binding:"required"`
	Amount         int64  `json:"amount"`      // บาทจำนวนเต็ม
	ContentType    string `json:"contentType"` // image/png, image/jpeg
	SlipBase64     string `json:"slipBase64" binding:"required"`
	CheckDuplicate *bool  `json:"checkDuplicate,omitempty"`
}

// GET /api/orders/:id/payment-intent
func (ctl *PaymentController) GetPaymentIntent(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	oid, err := strconv.Atoi(idStr)
	if err != nil || oid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	// โหลดออเดอร์
	var ord entity.Order
	if err := ctl.DB.First(&ord, uint(oid)).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "order not found"})
		return
	}
	// จำกัดสิทธิ์ เจ้าของออเดอร์เท่านั้น
	if ord.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// โหลดร้าน เพื่อเอา PromptPay จากตาราง restaurants
	var rest entity.Restaurant
	if err := ctl.DB.First(&rest, ord.RestaurantID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "restaurant not found"})
		return
	}

	pp := digitsOnly(rest.PromptPay)
	if pp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "restaurant has no PromptPay (mobile or citizen ID) set"})
		return
	}

	// ✅ ที่นี่กำหนดชัดเจนว่า ord.Total = "บาท"
	amountBaht := float64(ord.Total)
	totalSatang := int64(math.Round(amountBaht * 100.0))

	c.JSON(http.StatusOK, gin.H{
		"orderId":          ord.ID,
		"restaurantId":     rest.ID,
		"restaurantUserId": rest.UserID,

		// ใช้ PromptPay จากตาราง restaurants แทนเบอร์ของ owner
		"promptPayMobile": pp, // คงชื่อเดิมเพื่อความเข้ากันได้กับ FE
		"promptPay":       pp, // bonus: เผื่อ FE อยากใช้ชื่อคีย์ตรง ๆ

		// ใช้งานใน FE เวอร์ชันใหม่
		"amount": amountBaht, // บาท ตรง ๆ

		// เผื่อจุดอื่นยังพึ่งพาฟิลด์เก่าอยู่
		"totalBaht":   amountBaht,  // บาท
		"totalSatang": totalSatang, // สตางค์ (บาท*100)
	})
}

// GET /api/orders/:id/payment-summary
func (ctl *PaymentController) GetPaymentSummary(c *gin.Context) {
	v, ok := c.Get("userId")
	if !ok || v == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	idStr := c.Param("id")
	oid, err := strconv.Atoi(idStr)
	if err != nil || oid <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order id"})
		return
	}

	var ord entity.Order
	if err := ctl.DB.First(&ord, uint(oid)).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "order not found"})
		return
	}
	if ord.UserID != uid {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// หา payment ล่าสุดของออเดอร์นี้
	var pay entity.Payment
	if err := ctl.DB.Where("order_id = ?", ord.ID).Order("id desc").First(&pay).Error; err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, gorm.ErrRecordNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": "payment not found"})
		return
	}
	if pay.PaidAt == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment not completed"})
		return
	}

	txnId := ""
	if pay.TransRef != nil {
		txnId = *pay.TransRef
	}

	c.JSON(http.StatusOK, gin.H{
		"orderCode":  fmt.Sprintf("ORD-%d", ord.ID),
		"paidAmount": float64(pay.Amount),
		"currency":   "THB",
		"method":     "PromptPay",
		"paidAt":     pay.PaidAt, // ISO8601
		"txnId":      txnId,
	})
}

// POST /api/payments/verify-easyslip
func (ctl *PaymentController) VerifyEasySlip(c *gin.Context) {
	// 1) bind + validate
	var req verifySlipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_request"})
		return
	}
	if req.SlipBase64 == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "missing_base64"})
		return
	}
	if req.CheckDuplicate == nil {
		def := true
		req.CheckDuplicate = &def
	}
	if ctl.EasySlipToken == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "missing_easyslip_token"})
		return
	}

	// 2) โหลด order + payment ก่อน แล้วเช็ค already_paid
	var order entity.Order
	if err := ctl.DB.First(&order, uint(req.OrderID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": fmt.Sprintf("order %d not found", req.OrderID)})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "db_find_order_error"})
		}
		return
	}

	var p entity.Payment
	if err := ctl.DB.Where("order_id = ?", order.ID).First(&p).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			p = entity.Payment{OrderID: order.ID}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "db_find_payment_error"})
			return
		}
	}
	log.Printf("[VERIFY] order = %d paymentID = %d already_paid = %v", order.ID, p.ID, p.PaidAt != nil)

	// ถ้าจ่ายแล้ว ตัดจบทันที — ไม่เรียก EasySlip
	if p.PaidAt != nil {
		c.JSON(http.StatusConflict, gin.H{
			"success":   false,
			"error":     "already_paid",
			"paymentId": p.ID,
			"transRef":  p.TransRef,
			"paidAt":    p.PaidAt,
		})
		return
	}

	// 3) เตรียมรูปแล้วค่อยเรียก EasySlip
	b64, err := stripDataURLHeader(req.SlipBase64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_base64"})
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
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "easyslip_unreachable"})
		return
	}
	defer resp.Body.Close()

	// ===== OK (200) -> success flow =====
	if resp.StatusCode == http.StatusOK {
		var ok easySlipOKResp
		if err := json.NewDecoder(resp.Body).Decode(&ok); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "easyslip_decode_error"})
			return
		}

		rawBaht := ok.Data.Amount.Amount
		slipBahtInt := int64(math.Round(rawBaht)) // เก็บเป็น "บาทจำนวนเต็ม"
		matched := (req.Amount == 0) || (slipBahtInt == req.Amount)

		// ยอดไม่ตรง -> 400
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{
				"success":        false,
				"error":          "amount_mismatch",
				"expectedBaht":   float64(req.Amount),
				"expectedSatang": req.Amount * 100,
				"slipData": gin.H{
					"amountBaht":   rawBaht,
					"amountSatang": int64(math.Round(rawBaht * 100)),
					"date":         ok.Data.Date,
					"transRef":     ok.Data.TransRef,
				},
			})
			return
		}

		// ยอดตรง -> save
		if req.ContentType == "" || !strings.HasPrefix(req.ContentType, "image/") {
			req.ContentType = "image/*"
		}
		p.SlipBase64 = b64
		p.SlipContentType = req.ContentType
		p.Amount = slipBahtInt
		p.TransRef = &ok.Data.TransRef

		now := time.Now()
		p.PaidAt = &now
		if err := ctl.DB.Where("status_name = ?", "Paid").First(&paidStatus).Error; err == nil {
			p.PaymentStatusID = paidStatus.ID
		}

		if err := ctl.DB.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "db_save_payment_error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"matchedAmount":  true,
			"paymentId":      p.ID,
			"expectedBaht":   float64(req.Amount),
			"expectedSatang": req.Amount * 100,
			"slipData": gin.H{
				"amountBaht":   rawBaht,
				"amountSatang": int64(math.Round(rawBaht * 100)),
				"date":         ok.Data.Date,
				"transRef":     ok.Data.TransRef,
				"sender":       ok.Data.Sender,
				"receiver":     ok.Data.Receiver,
				"payload":      ok.Data.Payload,
			},
		})
		return
	}

	// ===== Non-200 -> error flow (ต้องตอบ JSON เสมอ) =====
	var ek easySlipErrResp
	if err := json.NewDecoder(resp.Body).Decode(&ek); err != nil {
		// ถ้า decode ไม่ได้ ให้ส่งสถานะเดิมจาก upstream
		c.JSON(resp.StatusCode, gin.H{"success": false, "error": "easyslip_error"})
		return
	}

	switch ek.Message {
	case "duplicate_slip":
		if ek.Data == nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "duplicate_slip"})
			return
		}
		dupBaht := ek.Data.Amount.Amount
		dupBahtInt := int64(math.Round(dupBaht))
		matched := (req.Amount == 0) || (dupBahtInt == req.Amount)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{
				"success":        false,
				"error":          "amount_mismatch",
				"expectedBaht":   float64(req.Amount),
				"expectedSatang": req.Amount * 100,
				"slipData": gin.H{
					"amountBaht":   dupBaht,
					"amountSatang": int64(math.Round(dupBaht * 100)),
					"date":         ek.Data.Date,
					"transRef":     ek.Data.TransRef,
				},
			})
			return
		}
		// treat as success (idempotent)
		if req.ContentType == "" || !strings.HasPrefix(req.ContentType, "image/") {
			req.ContentType = "image/*"
		}
		p.SlipBase64 = b64
		p.SlipContentType = req.ContentType
		p.Amount = dupBahtInt
		p.TransRef = &ek.Data.TransRef

		now := time.Now()
		p.PaidAt = &now
		if err := ctl.DB.Where("status_name = ?", "Paid").First(&paidStatus).Error; err == nil {
			p.PaymentStatusID = paidStatus.ID
		}
		if err := ctl.DB.Save(&p).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "db_save_payment_error"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"success":        true,
			"matchedAmount":  true,
			"paymentId":      p.ID,
			"expectedBaht":   float64(req.Amount),
			"expectedSatang": req.Amount * 100,
			"slipData": gin.H{
				"amountBaht":   dupBaht,
				"amountSatang": int64(math.Round(dupBaht * 100)),
				"date":         ek.Data.Date,
				"transRef":     ek.Data.TransRef,
			},
		})
		return

	case "qrcode_not_found":
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "qrcode_not_found"})
		return

	case "invalid_image":
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid_image"})
		return

	case "unauthorized":
		// ปกติควรเป็น 401 จาก upstream แต่เรา treat เป็น 502 (token ฝั่ง server ผิด/หมดอายุ)
		c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": "unauthorized"})
		return

	case "quota_exceeded":
		c.JSON(http.StatusTooManyRequests, gin.H{"success": false, "error": "quota_exceeded"})
		return

	default:
		// unknown error จาก EasySlip — ส่งต่อ status code + message
		if ek.Message == "" {
			ek.Message = "easyslip_error"
		}
		c.JSON(resp.StatusCode, gin.H{"success": false, "error": ek.Message})
		return
	}
}
