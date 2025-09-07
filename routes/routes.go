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

	// ===== Orders (ต้องล็อกอิน) =====
	orderCtrl := controllers.NewOrderController(db)
	orders := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		orders.POST("/orders", orderCtrl.Create)
		orders.GET("/orders/:id", orderCtrl.Detail)
		orders.GET("/profile/order", orderCtrl.ListForMe) // ตามที่ controller ใช้ path นี้อยู่
	}


	// ===== Reviews =====
	rv := controllers.NewReviewController(db)

	// Public: ดูรีวิวของร้าน
	r.GET("/restaurants/:id/reviews", rv.ListForRestaurant)

	// Protected: สร้าง/ดูของตัวเอง
	reviews := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reviews.POST("/reviews", rv.Create)
		reviews.GET("/profile/reviews", rv.ListForMe)
		reviews.GET("/reviews/:id", rv.DetailForMe) // owner only (เช็คใน controller)
	}


}
