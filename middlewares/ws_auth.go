// middlewares/ws_auth.go
package middlewares

import (
	"backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// WSAuthMiddleware ใช้ตรวจสอบ JWT จากทั้ง query และ header
func WSAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenStr string

		// 1) ลองอ่านจาก query ก่อน
		if t := c.Query("token"); t != "" {
			tokenStr = t
		} else {
			// 2) ถ้าไม่มี ลองอ่านจาก Header
			h := c.GetHeader("Authorization")
			if h != "" && strings.HasPrefix(h, "Bearer ") {
				tokenStr = strings.TrimPrefix(h, "Bearer ")
			}
		}

		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}

		// 3) Parse JWT
		claims := &utils.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// 4) เก็บ userId, role ลง context
		c.Set("userId", claims.UserID)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}
