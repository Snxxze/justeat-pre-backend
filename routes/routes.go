package routes

import (
	"backend/configs"
	"backend/controllers"
	"backend/middlewares"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.Use(middlewares.CORSMiddleware())
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	db := configs.DB()

	// Controllers
	authCtrl := controllers.NewAuthController(db)
	restCtrl := controllers.NewRestaurantController(db)
	orderCtrl := controllers.NewOrderController(db)
	riderCtrl := controllers.NewRiderController(db)
	adminCtrl := controllers.NewAdminController(db)
	appCtrl := controllers.NewRestaurantApplicationController(db)

	// Auth (public)
	a := r.Group("/auth")
	{
		a.POST("/register", authCtrl.Register)
		a.POST("/login", authCtrl.Login)
	}

	// Auth (protected)
	aAuth := a.Group("", middlewares.AuthMiddleware())
	{
		aAuth.GET("/me", authCtrl.Me)
		aAuth.PATCH("/me", authCtrl.UpdateMeRequest)
	}
	
	// Public/User
	r.GET("/restaurants", restCtrl.List)
	r.GET("/restaurants/:id", restCtrl.Detail)

	// ยื่นสมัครเปิดร้าน (ต้องล็อกอิน)
	r.POST("/restaurant-applications", middlewares.AuthMiddleware(), appCtrl.Apply)

	// Orders (user)
	u := r.Group("/", middlewares.AuthMiddleware())
	{
		u.POST("/orders", orderCtrl.Create)
		u.GET("/orders/:id", orderCtrl.Detail)
	}

	// Profile
	profile := r.Group("/profile", middlewares.AuthMiddleware())
	{
		profile.GET("/order", orderCtrl.ListForMe)
	}

	// Partner Restaurant (owner/admin)
	partnerRest := r.Group("/partner/restaurant", middlewares.AuthMiddleware("owner", "admin"))
	{
		partnerRest.GET("/dashboard", restCtrl.Dashboard) // ?restaurantId=
		partnerRest.GET("/order", restCtrl.Orders)        // ?restaurantId=
		partnerRest.GET("/menu", restCtrl.Menus)          // ?restaurantId=
		partnerRest.POST("/menu", restCtrl.CreateMenu)
		partnerRest.PATCH("/menu/:id", restCtrl.UpdateMenu)
		partnerRest.GET("/account", restCtrl.Account)     // ?restaurantId=
	}

	// Partner Rider (rider/admin)
	partnerRider := r.Group("/partner/rider", middlewares.AuthMiddleware("rider", "admin"))
	{
		partnerRider.GET("/dashboard", riderCtrl.Dashboard)
		partnerRider.GET("/work", riderCtrl.JobList)
		partnerRider.GET("/histories", riderCtrl.Histories)
		partnerRider.GET("/profile", riderCtrl.Profile)
		partnerRider.PATCH("/jobs/:id/finish", riderCtrl.FinishJob)
	}

	// Admin (admin only)
	admin := r.Group("/admin", middlewares.AuthMiddleware("admin"))
	{
		admin.GET("/dashboard", adminCtrl.Dashboard)
		admin.GET("/restaurant", adminCtrl.Restaurants)
		admin.GET("/report", adminCtrl.Reports)
		admin.GET("/rider", adminCtrl.Riders)
		admin.GET("/promotion", adminCtrl.Promotions)
		admin.POST("/promotion", adminCtrl.CreatePromotion)

		// อนุมัติ/ปฏิเสธใบสมัครเปิดร้าน
		admin.GET("/restaurant-applications", appCtrl.List)                     // ?status=pending
		admin.PATCH("/restaurant-applications/:id/approve", appCtrl.Approve)
		admin.PATCH("/restaurant-applications/:id/reject", appCtrl.Reject)
	}
}
