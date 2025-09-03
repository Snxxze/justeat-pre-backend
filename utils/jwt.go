package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims เป็น custom JWT claims ที่เราจะใช้ในระบบ
type Claims struct {
	UserID uint   `json:"userId"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken สร้าง JWT สำหรับผู้ใช้
func GenerateToken(userID uint, role string, secret string, ttl time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl)), // อายุ token
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
