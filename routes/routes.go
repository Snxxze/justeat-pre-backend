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
}
