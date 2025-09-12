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

	"log"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, cfg *configs.Config) {
	log.Printf("[ROUTES] EasySlip len=%d", len(cfg.EasySlipAPIKey))

	// ------------------------------------------------------------
	//Repositories
	// ------------------------------------------------------------
	userRepo := repository.NewUserRepository(db)
	cartRepo := repository.NewCartRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	chatRepo := repository.NewChatRepository(db)


	// ------------------------------------------------------------
	// Services
	// ------------------------------------------------------------
	authService := services.NewAuthService(userRepo, cfg.JWTSecret, cfg.JWTTTL)	
	userPromoService := services.NewUserPromotionService(db)

	cartSvc := services.NewCartService(db, cartRepo, orderRepo)

	chatService := services.NewChatService(db, chatRepo)

	// Hub WS
	hub := chatws.NewChatHub(chatService)
	go hub.Run()

	// ------------------------------------------------------------
	// Controllers
	// ------------------------------------------------------------
	authController := controllers.NewAuthController(authService)
	menuController := controllers.NewMenuController(db)
	reportController := controllers.NewReportController(db)
	rAppController := controllers.NewRestaurantApplicationController(db)
	riderAppCtl := controllers.NewRiderApplicationController(db)
	
	ownerOrderCtl := controllers.NewOwnerOrderController(db)
	cartCtl := controllers.NewCartController(cartSvc)
	riderCtl := controllers.NewRiderController(db)
	chatController := controllers.NewChatController(chatService)
	reviewCtl := controllers.NewReviewController(db)
	orderCtl := controllers.NewOrderController(db)
	restController := controllers.NewRestaurantController(db)
	
	userPromoCtrl := controllers.NewUserPromotionController(userPromoService)
	adminCtrl := controllers.NewAdminController(db)

	// ------------------------------------------------------------
	// Routes
	// ------------------------------------------------------------
	// ---------- Auth ----------
	authGroup := r.Group("/auth")
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
		riderGroup.GET("/works", riderCtl.ListWorks)
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

	// Payment controller
	paymentController := controllers.NewPaymentController(db, cfg.EasySlipAPIKey) // ===== EasySlip API Key ส่วนมากปัญหาอยู่ตรงนี้

	r.GET("/api/orders/:id/payment-intent", middlewares.AuthMiddleware(cfg.JWTSecret), paymentController.GetPaymentIntent)
	r.GET("/api/orders/:id/payment-summary", middlewares.AuthMiddleware(cfg.JWTSecret), paymentController.GetPaymentSummary)
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

	// ---------------- admin ---------------
	admin := r.Group("/admin")
	admin.Use(middlewares.AuthMiddleware(cfg.JWTSecret, "admin"))
	{
		admin.GET("/dashboard", adminCtrl.Dashboard)
		admin.GET("/restaurant", adminCtrl.Restaurants)
		admin.GET("/rider", adminCtrl.Riders)

		admin.GET("/reports", reportController.ListAllReports)
		admin.PATCH("reports/:id/status", reportController.UpdateReportStatus)
		admin.DELETE("/reports/:id", reportController.DeleteReport)

		// Promotion Management
		admin.GET("/promotion", adminCtrl.Promotions)
		admin.POST("/promotion", adminCtrl.CreatePromotion)
		admin.PUT("/promotion/:id", adminCtrl.UpdatePromotion)
		admin.DELETE("/promotion/:id", adminCtrl.DeletePromotion)
	}

	// ----------------- user -----------------
	user := r.Group("/user")
	user.Use(middlewares.AuthMiddleware(cfg.JWTSecret)) // ตรวจ JWT อย่างเดียว ไม่บังคับ role
	{
		user.GET("/promotions", userPromoCtrl.List)                // ดูรายการที่ user คนนั้นเก็บไว้
		user.POST("/promotions", userPromoCtrl.SavePromotion)      // body: { promoId } หรือ { promotionId }
		user.POST("/promotions/:id", userPromoCtrl.SavePromotion)  // หรือ path param
		user.POST("/promotions/:id/use", userPromoCtrl.UsePromotion)
	}

	// ---------- Public 
	r.GET("/promotions", controllers.ListActivePromotions(db))
	r.GET("/restaurants/:id/reviews", reviewCtl.ListForRestaurant)

	auth := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		auth.POST("/reviews", reviewCtl.Create)
		auth.GET("/profile/reviews", reviewCtl.ListForMe)
		auth.GET("/reviews/:id", reviewCtl.DetailForMe)
	}
}
