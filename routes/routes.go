package routes

import (
	"backend/configs"
	"backend/controllers"
	"backend/middlewares"
	"backend/repository"
	"backend/services"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"log"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *configs.Config) {
	log.Printf("[ROUTES] EasySlip len=%d", len(cfg.EasySlipAPIKey))
	//===== Auth =====
	// repo -> serviec -> controller
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authController := controllers.NewAuthController(authService)

	// Group: Auth
	authGroup := r.Group("/auth")
	{
		// Public route
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)

		// Protected
		authGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
		{
			authGroup.GET("/me", authController.Me)
			authGroup.PATCH("/me", authController.UpdateMe)
			authGroup.POST("/me/avatar", authController.UploadAvatar)
			authGroup.GET("/me/avatar", authController.GetAvatar)

		}
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

		// Payment controller
		paymentController := controllers.NewPaymentController(db, cfg.EasySlipAPIKey) // ===== EasySlip API Key ส่วนมากปัญหาอยู่ตรงนี้
		//  เพิ่ม API Group
		apiGroup := r.Group("/api")
		{
			// Payment endpoints
			paymentsGroup := apiGroup.Group("/payments")
			paymentsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
			{
				paymentsGroup.POST("/verify-easyslip", paymentController.VerifyEasySlip)
			}
		}
	}
}
