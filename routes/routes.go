package routes

import (
	"backend/configs"
	"backend/controllers"
	"backend/middlewares"
	"backend/repository"
	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg * configs.Config) {
	//===== Auth =====
	// repo -> serviec -> controller
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authController := controllers.NewAuthController(authService)

	// Payment controller
    paymentController := controllers.NewPaymentController(db)

	// Group: Auth
	auth := r.Group("/auth")
	{
		// Public route
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)

		// Protected
		auth.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
		{
			auth.GET("/me", authController.Me)
			auth.PATCH("/me", authController.UpdateMe)
			auth.POST("/me/avatar", authController.UploadAvatar)
			auth.GET("/me/avatar", authController.GetAvatar)
		}
	}

	// API group
    api := r.Group("/api")
    {
        // Payment routes
        payments := api.Group("/payments")
        {
            payments.POST("/upload-slip", paymentController.UploadSlip)
            // เพิ่ม routes อื่นๆ ที่จำเป็น
        }
        
        // ... routes อื่นๆ ของคุณ
    }

	// Reports
	reportRepo := repository.NewReportRepository(db)
	reportService := services.NewReportService(reportRepo)
	reportController := controllers.NewReportController(reportService)

	reports := r.Group("/reports")
	reports.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		// Create Report
		reports.POST("", reportController.CreateReport)

		// Get All Reports (ของ user)
		reports.GET("", reportController.ListReports)

		// ถ้าอยากดึงเฉพาะอันเดียว
		reports.GET("/:id", reportController.GetReportByID)
	}

	// ===== Payments (เก็บสลิป Base64) =====
	paymentCtl := controllers.NewPaymentController(db)

	payments := r.Group("/payments")
	payments.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		payments.POST("/upload-slip", paymentCtl.UploadSlip)
		// (ทางเลือก) แสดงสลิปกลับมาให้แอดมินดู
		// payments.GET("/:id/slip", paymentCtl.GetSlip)
	}
}
