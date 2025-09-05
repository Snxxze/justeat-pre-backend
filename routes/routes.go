package routes

import (
	"backend/configs"
	"backend/controllers"
	"backend/middlewares"
	"backend/repository"
	"backend/services"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"time"
)

// RegisterRoutes: ผูกทุกเส้นทางของแอป
func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *configs.Config) {
	// ----- CORS -----
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// ---------- Auth ----------
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authController := controllers.NewAuthController(authService)

	auth := r.Group("/auth")
	{
		// public
		auth.POST("/register", authController.Register)
		auth.POST("/login", authController.Login)

		// protected
		auth.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
		{
			auth.GET("/me", authController.Me)
			auth.PATCH("/me", authController.UpdateMe)
			auth.POST("/me/avatar", authController.UploadAvatar)
			auth.GET("/me/avatar", authController.GetAvatar)
		}
	}

	// ---------- Reports (ต้องล็อกอิน) ----------
	reportRepo := repository.NewReportRepository(db)
	reportService := services.NewReportService(reportRepo)
	reportController := controllers.NewReportController(reportService)

	reports := r.Group("/reports")
	reports.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reports.POST("", reportController.CreateReport)
		reports.GET("", reportController.ListReports)
		reports.GET("/:id", reportController.GetReportByID)
	}

	// ---------- Admin (ต้องล็อกอิน / ถ้าจะบังคับ role=admin ให้ใส่ middlewares.AuthMiddleware(cfg.JWTSecret, "admin")) ----------
	adminCtrl := controllers.NewAdminController(db)
	appCtrl := controllers.NewRestaurantApplicationController(db)

	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		admin.GET("/dashboard", adminCtrl.Dashboard)
		admin.GET("/restaurant", adminCtrl.Restaurants)
		admin.GET("/report", adminCtrl.Reports)
		admin.GET("/rider", adminCtrl.Riders)

		// Promotion Management
		admin.GET("/promotion", adminCtrl.Promotions)
		admin.POST("/promotion", adminCtrl.CreatePromotion)
		admin.PUT("/promotion/:id", adminCtrl.UpdatePromotion)
		admin.DELETE("/promotion/:id", adminCtrl.DeletePromotion)

		// Restaurant Applications
		admin.GET("/restaurant-applications", appCtrl.List)
		admin.PATCH("/restaurant-applications/:id/approve", appCtrl.Approve)
		admin.PATCH("/restaurant-applications/:id/reject", appCtrl.Reject)
	}

	// ---------- User Promotions (ต้องล็อกอิน / ทุก role ใช้ได้) ----------
	userPromoService := services.NewUserPromotionService(db)
	userPromoCtrl := controllers.NewUserPromotionController(userPromoService)

	user := r.Group("/user")
	user.Use(middlewares.AuthMiddleware(cfg.JWTSecret)) // ตรวจ JWT อย่างเดียว ไม่บังคับ role
	{
		user.GET("/promotions", userPromoCtrl.List)                // ดูรายการที่ user คนนั้นเก็บไว้
		user.POST("/promotions", userPromoCtrl.SavePromotion)      // body: { promoId } หรือ { promotionId }
		user.POST("/promotions/:id", userPromoCtrl.SavePromotion)  // หรือ path param
		user.POST("/promotions/:id/use", userPromoCtrl.UsePromotion)
	}

	// ---------- Public Promotions (ไม่ต้องล็อกอิน) ----------
	r.GET("/promotions", controllers.ListActivePromotions(db))

	// // (ตัวช่วย debug เส้นทาง - ใช้ชั่วคราวแล้วคอมเมนต์ทิ้ง)
	// for _, ri := range r.Routes() {
	// 	log.Printf("[ROUTE] %s %s", ri.Method, ri.Path)
	// }
}
