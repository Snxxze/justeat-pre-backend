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

	// ------------------------------------------------------------
	// Repositories
	// ------------------------------------------------------------
	userRepo := repository.NewUserRepository(db)
	restRepo := repository.NewRestaurantRepository(db)
	menuRepo := repository.NewMenuRepository(db)
	reportRepo := repository.NewReportRepository(db)
	rAppRepo := repository.NewRestaurantApplicationRepository(db)
	chatRepo := repository.NewChatRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	riderRepo := repository.NewRiderRepository(db)
	riderWorkRepo := repository.NewRiderWorkRepository(db)
	riderAppRepo := repository.NewRiderApplicationRepository(db)

	// ------------------------------------------------------------
	// Services
	// ------------------------------------------------------------
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	restService := services.NewRestaurantService(restRepo)
	menuService := services.NewMenuService(menuRepo)
	reportService := services.NewReportService(reportRepo)
	rAppService := services.NewRestaurantApplicationService(rAppRepo)
	chatService := services.NewChatService(chatRepo)
	riderAppSvc  := services.NewRiderApplicationService(riderAppRepo ,riderRepo)

	// Order/Cart
	orderSvc := services.NewOrderService(db, orderRepo, cartRepo, restRepo)
	cartSvc := services.NewCartService(db, cartRepo, orderRepo)
	riderSvc := services.NewRiderService(db, riderRepo, riderWorkRepo, orderRepo)

	// ------------------------------------------------------------
	// Controllers
	// ------------------------------------------------------------
	authController := controllers.NewAuthController(authService)
	restController := controllers.NewRestaurantController(restService)
	menuController := controllers.NewMenuController(menuService)
	reportController := controllers.NewReportController(reportService)
	rAppController := controllers.NewRestaurantApplicationController(rAppService)
	chatController := controllers.NewChatController(chatService)
	riderAppCtl  := controllers.NewRiderApplicationController(riderAppSvc)

	orderCtl := controllers.NewOrderController(orderSvc)
	ownerOrderCtl := controllers.NewOwnerOrderController(orderSvc)
	cartCtl := controllers.NewCartController(cartSvc)

	riderCtl := controllers.NewRiderController(riderSvc)

	// ------------------------------------------------------------
	// Routes
	// ------------------------------------------------------------

	// ---------- Auth ----------
	authGroup := r.Group("/auth")
	{
		// Public
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
	reportsGroup := r.Group("/reports")
	reportsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reportsGroup.POST("", reportController.CreateReport)     // Create Report
		reportsGroup.GET("", reportController.ListReports)       // Get All Reports ของ user
		reportsGroup.GET("/:id", reportController.GetReportByID) // Get Report by ID
	}

	// ---------- Restaurants ----------
	// Public
	r.GET("/restaurants", restController.List)
	r.GET("/restaurants/:id", restController.Get)

	// Public: ลูกค้าเห็นเมนูของร้าน
	r.GET("/restaurants/:id/menus", menuController.ListByRestaurant)
	r.GET("/menus/:id", menuController.Get)

	// ---------- Owner ----------
	ownerGroup := r.Group("/owner")
	ownerGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		ownerGroup.GET("/restaurants/:id/orders", ownerOrderCtl.List)
		ownerGroup.GET("/restaurants/:id/orders/:orderId", ownerOrderCtl.Detail)
		ownerGroup.PATCH("/restaurants/:id", restController.Update)

		ownerGroup.POST("/restaurants/:id/menus", menuController.Create)
		ownerGroup.PATCH("/menus/:id", menuController.Update)
		ownerGroup.DELETE("/menus/:id", menuController.Delete)
		ownerGroup.PATCH("/menus/:id/status", menuController.UpdateStatus)

		ownerGroup.POST("/orders/:orderId/accept", ownerOrderCtl.Accept) // Pending -> Preparing
		ownerGroup.POST("/orders/:orderId/cancel", ownerOrderCtl.Cancel) // Pending -> Cancelled
	}

	// ---------- Rider ----------
	riderGroup := r.Group("/rider", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		riderGroup.PATCH("/me/availability", riderCtl.SetAvailability) // ONLINE / OFFLINE
		riderGroup.GET("/works/available", riderCtl.ListAvailable) 
		
		riderGroup.POST("/works/:orderId/accept", riderCtl.Accept)     // Preparing -> Delivering (assign งาน)
		riderGroup.POST("/works/:orderId/complete", riderCtl.Complete) // Delivering -> Completed
	}

	// ---------- Restaurant Applications ----------
	partnerRestApps := r.Group("/partner/restaurant-applications")
	partnerRestApps.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		partnerRestApps.POST("", rAppController.Apply)                // ยื่นสมัคร
		partnerRestApps.GET("", rAppController.List)                  // แอดมินดูรายการ
	}

	adminRestApps := r.Group("/partner/restaurant-applications", middlewares.AuthMiddleware(cfg.JWTSecret, "admin")) 
	{
		adminRestApps.PATCH("/:id/approve", rAppController.Approve) // อนุมัติ
		adminRestApps.PATCH("/:id/reject", rAppController.Reject)   // ปฏิเสธ
	}

	// ---------- Rider Applications ----------
	// ผู้ใช้ยื่น/ดูของตัวเอง
	userRiderApps := r.Group("/partner/rider-applications", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		userRiderApps.POST("", riderAppCtl.Apply)
		userRiderApps.GET("/mine", riderAppCtl.ListMine)
	}

	// แอดมินจัดการ
	adminRiderApps := r.Group("/partner/rider-applications", middlewares.AuthMiddleware(cfg.JWTSecret, "admin"))
	{
		adminRiderApps.GET("", riderAppCtl.List)
		adminRiderApps.PATCH("/:id/approve", riderAppCtl.Approve)
		adminRiderApps.PATCH("/:id/reject", riderAppCtl.Reject)
	}

	// ---------- Chat ----------
	chatGroup := r.Group("/chatrooms", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		chatGroup.GET("", chatController.ListRooms)
		chatGroup.GET("/:id/messages", chatController.ListMessages)
		chatGroup.POST("/:id/messages", chatController.SendMessage)
	}

	// ---------- Cart / Orders (ลูกค้า) ----------
	authOrder := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		authOrder.POST("/orders", orderCtl.Create)
		authOrder.GET("/profile/orders", orderCtl.ListForMe)
		authOrder.GET("/orders/:id", orderCtl.Detail)

		authOrder.GET("/cart", cartCtl.Get)
		authOrder.POST("/cart/items", cartCtl.Add)
		authOrder.PATCH("/cart/items/qty", cartCtl.UpdateQty)
		authOrder.DELETE("/cart/items", cartCtl.RemoveItem)
		authOrder.DELETE("/cart", cartCtl.Clear)
		authOrder.POST("/orders/checkout-from-cart", orderCtl.CheckoutFromCart)
	}
}
