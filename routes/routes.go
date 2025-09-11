package routes

import (
	"backend/configs"
	"backend/controllers"
	"backend/middlewares"
	"backend/repository"
	"backend/services"
	chatws "backend/ws"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg * configs.Config) {
	//===== Auth =====
	// repo -> serviec -> controller
	userRepo := repository.NewUserRepository(db)
	restRepo := repository.NewRestaurantRepository(db)
	menuRepo := repository.NewMenuRepository(db)
	reportRepo := repository.NewReportRepository(db)
	rAppRepo := repository.NewRestaurantApplicationRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	riderRepo := repository.NewRiderRepository(db)
	riderWorkRepo := repository.NewRiderWorkRepository(db)
	riderAppRepo := repository.NewRiderApplicationRepository(db)
	chatRepo := repository.NewChatRepository(db)

	// ------------------------------------------------------------
	// Services
	// ------------------------------------------------------------
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)
	restService := services.NewRestaurantService(restRepo)
	menuService := services.NewMenuService(menuRepo)
	reportService := services.NewReportService(reportRepo)
	rAppService := services.NewRestaurantApplicationService(rAppRepo)
	riderAppSvc := services.NewRiderApplicationService(riderAppRepo, riderRepo)

	orderSvc := services.NewOrderService(db, orderRepo, cartRepo, restRepo)
	cartSvc := services.NewCartService(db, cartRepo, orderRepo)
	riderSvc := services.NewRiderService(db, riderRepo, riderWorkRepo, orderRepo)
	chatService := services.NewChatService(db, chatRepo)

	// Hub WS
	hub := chatws.NewChatHub(chatService)
	go hub.Run()

	// ------------------------------------------------------------
	// Controllers
	// ------------------------------------------------------------
	authController := controllers.NewAuthController(authService)
	restController := controllers.NewRestaurantController(restService)
	menuController := controllers.NewMenuController(menuService)
	reportController := controllers.NewReportController(reportService)
	rAppController := controllers.NewRestaurantApplicationController(rAppService)
	riderAppCtl := controllers.NewRiderApplicationController(riderAppSvc)

	orderCtl := controllers.NewOrderController(orderSvc)
	ownerOrderCtl := controllers.NewOwnerOrderController(orderSvc)
	cartCtl := controllers.NewCartController(cartSvc)
	riderCtl := controllers.NewRiderController(riderSvc)
	chatController := controllers.NewChatController(chatService)

	// Group: Auth
	auth := r.Group("/auth")
	{
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)

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
	reportsGroup := r.Group("/reports", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reportsGroup.POST("", reportController.CreateReport)
		reportsGroup.GET("", reportController.ListReports)
		reportsGroup.GET("/:id", reportController.GetReportByID)
	}

	// ---------- Restaurants ----------
	r.GET("/restaurants", restController.List)
	r.GET("/restaurants/:id", restController.Get)
	r.GET("/restaurants/:id/menus", menuController.ListByRestaurant)
	r.GET("/menus/:id", menuController.Get)

	// ---------- Owner ----------
	ownerGroup := r.Group("/owner", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		ownerGroup.GET("/restaurants/:id/orders", ownerOrderCtl.List)
		ownerGroup.GET("/restaurants/:id/orders/:orderId", ownerOrderCtl.Detail)
		ownerGroup.PATCH("/restaurants/:id", restController.Update)
		ownerGroup.POST("/restaurants/:id/menus", menuController.Create)
		ownerGroup.PATCH("/menus/:id", menuController.Update)
		ownerGroup.DELETE("/menus/:id", menuController.Delete)
		ownerGroup.PATCH("/menus/:id/status", menuController.UpdateStatus)
		ownerGroup.POST("/orders/:orderId/accept", ownerOrderCtl.Accept)
		ownerGroup.POST("/orders/:orderId/cancel", ownerOrderCtl.Cancel)
	}

	// ---------- Rider ----------
	riderGroup := r.Group("/rider", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		riderGroup.PATCH("/me/availability", riderCtl.SetAvailability)
		riderGroup.GET("/me/status", riderCtl.GetStatus)
		riderGroup.GET("/works/current", riderCtl.GetCurrentWork)
		riderGroup.GET("/works/available", riderCtl.ListAvailable)
		riderGroup.POST("/works/:orderId/accept", riderCtl.Accept)
		riderGroup.POST("/works/:orderId/complete", riderCtl.Complete)
	}

	// ---------- Restaurant Applications ----------
	partnerRestApps := r.Group("/partner/restaurant-applications", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		partnerRestApps.POST("", rAppController.Apply)
		partnerRestApps.GET("", rAppController.List)
	}

	adminRestApps := r.Group("/partner/restaurant-applications", middlewares.AuthMiddleware(cfg.JWTSecret, "admin"))
	{
		adminRestApps.PATCH("/:id/approve", rAppController.Approve)
		adminRestApps.PATCH("/:id/reject", rAppController.Reject)
	}

	// ---------- Rider Applications ----------
	userRiderApps := r.Group("/partner/rider-applications", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		userRiderApps.POST("", riderAppCtl.Apply)
		userRiderApps.GET("/mine", riderAppCtl.ListMine)
	}

	adminRiderApps := r.Group("/partner/rider-applications", middlewares.AuthMiddleware(cfg.JWTSecret, "admin"))
	{
		adminRiderApps.GET("", riderAppCtl.List)
		adminRiderApps.PATCH("/:id/approve", riderAppCtl.Approve)
		adminRiderApps.PATCH("/:id/reject", riderAppCtl.Reject)
	}

	// ---------- Orders + Cart ----------
	authOrder := r.Group("/orders", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		authOrder.POST("", orderCtl.Create)
		authOrder.GET("/profile", orderCtl.ListForMe)
		authOrder.GET("/:id", orderCtl.Detail)
		authOrder.POST("/checkout-from-cart", orderCtl.CheckoutFromCart)

		// Chat REST
		authOrder.GET("/:id/chatroom", chatController.GetOrCreateRoom)
		authOrder.GET("/:id/messages", chatController.GetMessages)
		authOrder.POST("/:id/messages", chatController.SendMessage)
	}

	authCart := r.Group("/cart", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		authCart.GET("", cartCtl.Get)
		authCart.POST("/items", cartCtl.Add)
		authCart.PATCH("/items/qty", cartCtl.UpdateQty)
		authCart.DELETE("/items", cartCtl.RemoveItem)
		authCart.DELETE("", cartCtl.Clear)
	}

	// ---------- Chat WS ----------
	wsGroup := r.Group("/ws", middlewares.WSAuthMiddleware(cfg.JWTSecret))
	{
		wsGroup.GET("/chat/:roomId", hub.HandleWebSocket)
	}

	// ---------- Payment ----------
	paymentController := controllers.NewPaymentController(db, cfg.EasySlipAPIKey)
	apiGroup := r.Group("/api")
	{
		paymentsGroup := apiGroup.Group("/payments", middlewares.AuthMiddleware(cfg.JWTSecret))
		{
			paymentsGroup.POST("/verify-easyslip", paymentController.VerifyEasySlip)
		}
	}
}
