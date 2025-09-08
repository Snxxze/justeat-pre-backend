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
	menuOptRepo := repository.NewMenuOptionRepository(db)
	optRepo := repository.NewOptionRepository(db)
	reportRepo := repository.NewReportRepository(db)
	rAppRepo := repository.NewRestaurantApplicationRepository(db)
	chatRepo := repository.NewChatRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)

	// ------------------------------------------------------------
	// Services
	// ------------------------------------------------------------
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	restService := services.NewRestaurantService(restRepo)
	menuService := services.NewMenuService(menuRepo)
	menuOptService := services.NewMenuOptionService(menuOptRepo)
	optService := services.NewOptionService(optRepo)
	reportService := services.NewReportService(reportRepo)
	rAppService := services.NewRestaurantApplicationService(rAppRepo)
	chatService := services.NewChatService(chatRepo)

	// Order/Cart (ตามเดิม)
	orderSvc := services.NewOrderService(db, orderRepo, cartRepo, restRepo)
	cartSvc := services.NewCartService(db, cartRepo, orderRepo)

	// ------------------------------------------------------------
	// Controllers
	// ------------------------------------------------------------
	authController := controllers.NewAuthController(authService)
	restController := controllers.NewRestaurantController(restService)
	menuController := controllers.NewMenuController(menuService)
	menuOptController := controllers.NewMenuOptionController(menuOptService)
	optController := controllers.NewOptionController(optService)
	reportController := controllers.NewReportController(reportService)
	rAppController := controllers.NewRestaurantApplicationController(rAppService)
	chatController := controllers.NewChatController(chatService)

	orderCtl := controllers.NewOrderController(orderSvc)
	ownerOrderCtl := controllers.NewOwnerOrderController(orderSvc)
	cartCtl := controllers.NewCartController(cartSvc)

	// ------------------------------------------------------------
	// Routes
	// ------------------------------------------------------------

	// ---------- Auth ----------
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
	reportsGroup := r.Group("/reports")
	reportsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reportsGroup.POST("", reportController.CreateReport)        // Create Report
		reportsGroup.GET("", reportController.ListReports)          // Get All Reports ของ user
		reportsGroup.GET("/:id", reportController.GetReportByID)    // Get Report by ID
	}

	// ---------- Restaurants ----------
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

		// Option
		ownerGroup.POST("/options", optController.Create)
		ownerGroup.PATCH("/options/:id", optController.Update)
		ownerGroup.DELETE("/options/:id", optController.Delete)

		ownerGroup.POST("/menus/:id/options", menuOptController.AttachOption)
		ownerGroup.DELETE("/menus/:id/options/:optionId", menuOptController.DetachOption)
	}

	// ---------- Restaurant Applications ----------
	appsGroup := r.Group("/partner/restaurant-applications")
	appsGroup.Use(middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		appsGroup.POST("", rAppController.Apply)                 // ยื่นสมัคร
		appsGroup.GET("", rAppController.List)                   // แอดมินดูรายการ
		appsGroup.PATCH("/:id/approve", rAppController.Approve)  // อนุมัติ
		appsGroup.PATCH("/:id/reject", rAppController.Reject)    // ปฏิเสธ
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
