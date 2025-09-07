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

	// ---------- Menu Options ----------
	menuOptRepo := repository.NewMenuOptionRepository(db)
	menuOptService := services.NewMenuOptionService(menuOptRepo)
	menuOptController := controllers.NewMenuOptionController(menuOptService)

	// ---------- Options ----------
	optRepo := repository.NewOptionRepository(db)
	optService := services.NewOptionService(optRepo)
	optController := controllers.NewOptionController(optService)

	// Public
	r.GET("/restaurants", restController.List)
	r.GET("/restaurants/:id", restController.Get)

	// Public: ลูกค้าเห็นเมนูของร้าน
	r.GET("/restaurants/:id/menus", menuController.ListByRestaurant)
	r.GET("/menus/:id", menuController.Get)
	// Public: ลูกค้าดู options ของเมนูได้
	r.GET("/menus/:id/options", menuOptController.ListByMenu)
	
	// ลูกค้า: ดู option ได้ (public)
	r.GET("/options", optController.List)
	r.GET("/options/:id", optController.Get)

	// Owner
	ownerGroup := r.Group("/owner")
	ownerGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		ownerGroup.PATCH("/restaurants/:id", restController.Update)
		ownerGroup.POST("/restaurants/:id/menus", menuController.Create)
    ownerGroup.PATCH("/menus/:id", menuController.Update)
    ownerGroup.DELETE("/menus/:id", menuController.Delete)

		ownerGroup.PATCH("/menus/:id/status", menuController.UpdateStatus)

		// Option
    ownerGroup.POST("/options", optController.Create)
		ownerGroup.PATCH("/options/:id", optController.Update)
		ownerGroup.DELETE("/options/:id", optController.Delete)

		ownerGroup.POST("/menus/:id/options", menuOptController.AttachOption)
		ownerGroup.DELETE("/menus/:id/options/:optionId", menuOptController.DetachOption)
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

	orderRepo := repository.NewOrderRepository(db)
	orderSvc  := services.NewOrderService(db, orderRepo)
	orderCtl  := controllers.NewOrderController(orderSvc)

	authOrder := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret)) // ใช้ middleware เดิมของคุณ
	{
		authOrder.POST("/orders", orderCtl.Create)
		authOrder.GET("/profile/orders", orderCtl.ListForMe)
		authOrder.GET("/orders/:id", orderCtl.Detail)
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
