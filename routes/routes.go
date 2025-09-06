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

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *configs.Config) {

	// ---------- Auth ----------
	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	authController := controllers.NewAuthController(authService)

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

			authGroup.GET("/me/restaurant", authController.MeRestaurant)
		}
	}

	// ---------- Reports ----------
	reportRepo := repository.NewReportRepository(db)
	reportService := services.NewReportService(reportRepo)
	reportController := controllers.NewReportController(reportService)

	reportsGroup := r.Group("/reports")
	reportsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reportsGroup.POST("", reportController.CreateReport)   // Create Report
		reportsGroup.GET("", reportController.ListReports)     // Get All Reports ของ user
		reportsGroup.GET("/:id", reportController.GetReportByID) // Get Report by ID
	}

	// ---------- Restaurants ----------
	restRepo := repository.NewRestaurantRepository(db)
	restService := services.NewRestaurantService(restRepo)
	restController := controllers.NewRestaurantController(restService)

	menuRepo := repository.NewMenuRepository(db)
	menuService := services.NewMenuService(menuRepo)
	menuController := controllers.NewMenuController(menuService)

	// Public
	r.GET("/restaurants", restController.List)
	r.GET("/restaurants/:id", restController.Get)

	// Public: ลูกค้าเห็นเมนูของร้าน
	r.GET("/restaurants/:id/menus", menuController.ListByRestaurant)
	r.GET("/menus/:id", menuController.Get)

	// Owner
	ownerGroup := r.Group("/owner")
	ownerGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		ownerGroup.PATCH("/restaurants/:id", restController.Update)
		ownerGroup.POST("/restaurants/:id/menus", menuController.Create)
    ownerGroup.PATCH("/menus/:id", menuController.Update)
    ownerGroup.DELETE("/menus/:id", menuController.Delete)
	}

	// ---------- Restaurant Applications ----------
	rAppRepo := repository.NewRestaurantApplicationRepository(db)
	rAppService := services.NewRestaurantApplicationService(rAppRepo)
	rAppController := controllers.NewRestaurantApplicationController(rAppService)

	appsGroup := r.Group("/partner/restaurant-applications")
	appsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		appsGroup.POST("", rAppController.Apply)              // ยื่นสมัคร
		appsGroup.GET("", rAppController.List)                // แอดมินดูรายการ
		appsGroup.PATCH("/:id/approve", rAppController.Approve) // อนุมัติ
		appsGroup.PATCH("/:id/reject", rAppController.Reject)   // ปฏิเสธ
	}

	chatRepo := repository.NewChatRepository(db)
	chatService := services.NewChatService(chatRepo)
	chatController := controllers.NewChatController(chatService)

	auth := r.Group("/chatrooms", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		auth.GET("", chatController.ListRooms)
		auth.GET("/:id/messages", chatController.ListMessages)
		auth.POST("/:id/messages", chatController.SendMessage)
	}
}
