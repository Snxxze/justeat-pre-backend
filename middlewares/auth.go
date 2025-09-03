package middlewares

import (
	"backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AuthMiddleware ตรวจสอบ JWT และบังคับ role (ถ้ามีการกำหนด)
func AuthMiddleware(secret string, requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// ---------------- ตรวจ Header ----------------
		h := c.GetHeader("Authorization")
		if h == "" || !strings.HasPrefix(h, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "missing or invalid token"})
			c.Abort()
			return
		}
		tokenStr := strings.TrimPrefix(h, "Bearer ")

		// ---------------- Parse JWT ----------------
		claims := &utils.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid token"})
			c.Abort()
			return
		}

		// ---------------- Extract Claims ----------------
		if claims.UserID == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"ok": false, "error": "invalid userId"})
			c.Abort()
			return
		}

		// set ค่าไว้ใน context
		c.Set("userId", claims.UserID)
		c.Set("role", claims.Role)

		// ---------------- Role Checking ----------------
		if len(requiredRoles) > 0 {
			allowed := false
			for _, r := range requiredRoles {
				if claims.Role == r {
					allowed = true
					break
				}
			}
			if !allowed {
				c.JSON(http.StatusForbidden, gin.H{"ok": false, "error": "forbidden"})
				c.Abort()
				return
			}
		}

		// ผ่านทั้งหมด → ไป handler ต่อ
		c.Next()
	}
}
