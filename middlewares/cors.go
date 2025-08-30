package middlewares

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	cfg := cors.Config{
		AllowOrigins:  []string{"*"}, // dev เท่านั้น; prod ใส่โดเมนจริง
		AllowMethods:  []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Authorization", "Content-Type"},
		ExposeHeaders: []string{"Content-Length"},
		MaxAge:        12 * time.Hour,
	}
	return cors.New(cfg)
}
