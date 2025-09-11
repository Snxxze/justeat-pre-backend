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
	reportRepo := repository.NewReportRepository(db)
	reportService := services.NewReportService(reportRepo)
	reportController := controllers.NewReportController(reportService)

	reportsGroup := r.Group("/reports", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		reportsGroup.POST("", reportController.CreateReport)
		reportsGroup.GET("", reportController.ListReports)
		reportsGroup.GET("/:id", reportController.GetReportByID)
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
	r.GET("/restaurants/:id/menus", menuController.ListByRestaurant)
	r.GET("/menus/:id", menuController.Get)
	r.GET("/menus/:id/options", menuOptController.ListByMenu)
	r.GET("/options", optController.List)
	r.GET("/options/:id", optController.Get)

	// ---------- Owner (ล็อกให้ role=owner) 👇
	ownerGroup := r.Group("/owner", middlewares.AuthMiddleware(cfg.JWTSecret, "owner"))
	{
		ownerGroup.PATCH("/restaurants/:id", restController.Update)
		ownerGroup.POST("/restaurants/:id/menus", menuController.Create)
		ownerGroup.PATCH("/menus/:id", menuController.Update)
		ownerGroup.DELETE("/menus/:id", menuController.Delete)
		ownerGroup.PATCH("/menus/:id/status", menuController.UpdateStatus)

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

	// ผู้ใช้ยื่น (ต้อง login แต่ไม่ต้องเป็น admin) 👇
	userRestaurantApps := r.Group("/partner/restaurant-applications",
		middlewares.AuthMiddleware(cfg.JWTSecret),
	)
	{
		userRestaurantApps.POST("", rAppController.Apply)
	}

	// แอดมินจัดการ (ต้อง role=admin) 👇
	adminRestaurantApps := r.Group("/partner/restaurant-applications",
		middlewares.AuthMiddleware(cfg.JWTSecret, "admin"),
	)
	{
		adminRestaurantApps.GET("", rAppController.List)
		adminRestaurantApps.PATCH("/:id/approve", rAppController.Approve)
		adminRestaurantApps.PATCH("/:id/reject", rAppController.Reject)
	}

	

	// ---------- Chat ----------
	chatRepo := repository.NewChatRepository(db)
	chatService := services.NewChatService(chatRepo)
	chatController := controllers.NewChatController(chatService)

	chat := r.Group("/chatrooms", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		chat.GET("", chatController.ListRooms)
		chat.GET("/:id/messages", chatController.ListMessages)
		chat.POST("/:id/messages", chatController.SendMessage)
	}

	// ---------- Orders ----------
	orderRepo := repository.NewOrderRepository(db)
	orderSvc := services.NewOrderService(db, orderRepo)
	orderCtl := controllers.NewOrderController(orderSvc)

	authOrder := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		authOrder.POST("/orders", orderCtl.Create)
		authOrder.GET("/profile/orders", orderCtl.ListForMe)
		authOrder.GET("/orders/:id", orderCtl.Detail)
	}

	// ---------- Rider Applications ----------
	riderAppRepo := repository.NewRiderApplicationRepository(db)
	riderAppSvc  := services.NewRiderApplicationService(riderAppRepo)
	riderAppCtl  := controllers.NewRiderApplicationController(riderAppSvc)

	// ผู้ใช้ยื่น (login ธรรมดา) 👇
	userRiderApps := r.Group("/partner/rider-applications",
		middlewares.AuthMiddleware(cfg.JWTSecret),
	)
	{
		userRiderApps.POST("", riderAppCtl.Apply)
		userRiderApps.GET("/mine", riderAppCtl.ListMine)
	}

	// แอดมินจัดการ (role=admin) 👇
	adminRiderApps := r.Group("/partner/rider-applications",
		middlewares.AuthMiddleware(cfg.JWTSecret, "admin"),
	)
	{
		adminRiderApps.GET("", riderAppCtl.List)
		adminRiderApps.PATCH("/:id/approve", riderAppCtl.Approve)
		adminRiderApps.PATCH("/:id/reject", riderAppCtl.Reject)
	}

	// ---------- Reviews ----------
	// สร้าง controller
	rv := controllers.NewReviewController(db)

	// Public
	r.GET("/restaurants/:id/reviews", rv.ListForRestaurant)

	// Protected
	auth := r.Group("/", middlewares.AuthMiddleware(cfg.JWTSecret))
	{
		auth.POST("/reviews", rv.Create)
		auth.GET("/profile/reviews", rv.ListForMe) // <- ตัวนี้ต้องชี้มาที่เมธอดที่เรามีจริง
		auth.GET("/reviews/:id", rv.DetailForMe)
	}

}
