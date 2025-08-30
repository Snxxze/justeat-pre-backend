package middlewares

import (
	"fmt"
	"net/http"
	"strings"

	"backend/configs"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// ใช้ตรวจ token และ (ถ้ามี) บังคับ role
func AuthMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "missing or invalid token"})
			c.Abort(); return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")

		cfg := configs.LoadConfig()
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid token"})
			c.Abort(); return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid claims"})
			c.Abort(); return
		}

		var role string
		if v, ok := claims["role"].(string); ok {
			role = v
		}
		var userId uint
		switch v := claims["userId"].(type) {
		case float64:
			userId = uint(v)
		case int:
			userId = uint(v)
		case int64:
			userId = uint(v)
		case uint:
			userId = v
		}

		c.Set("userId", userId)
		c.Set("role", role)

		if len(requiredRoles) > 0 {
			allowed := false
			for _, r := range requiredRoles {
				if role == r { allowed = true; break }
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "forbidden"})
				c.Abort(); return
			}
		}

		c.Next()
	}
}
