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

	// repo -> serviec -> controller
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authController := controllers.NewAuthController(authService)

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

	// ---------- Restaurant ----------
	restRepo := repository.NewRestaurantRepository(db)
	restService := services.NewRestaurantService(restRepo)
	restController := controllers.NewRestaurantController(restService)

	partner := r.Group("/partner/restaurant")
	partner.Use(middlewares.AuthMiddleware(cfg.JWTSecret)) // ป้องกันเฉพาะ partner
	{
		partner.GET("/menu", restController.Menus)
		partner.POST("/menu", restController.CreateMenu)
		partner.PATCH("/menu/:id", restController.UpdateMenu)

		partner.GET("/dashboard", restController.Dashboard)
	}

	// ---------- Restaurant Applications ----------
	RAppRepo := repository.NewRestaurantApplicationRepository(db)
	RAppService := services.NewRestaurantApplicationService(RAppRepo)
	RAController := controllers.NewRestaurantApplicationController(RAppService)

	apps := r.Group("/partner/restaurant-applications")
	apps.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		apps.POST("", RAController.Apply)           // ยื่นสมัคร
		apps.GET("", RAController.List)             // แอดมินดูรายการ
		apps.PATCH("/:id/approve", RAController.Approve) // อนุมัติ
		apps.PATCH("/:id/reject", RAController.Reject)   // ปฏิเสธ
	}
}
