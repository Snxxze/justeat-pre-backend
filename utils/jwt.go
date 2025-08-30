package utils

import (
	"time"

	"backend/configs"
	"backend/entity"
	"github.com/golang-jwt/jwt/v5"
)

const AccessTokenTTL = 72 * time.Hour

func GenerateToken(user entity.User) (string, error) {
	cfg := configs.LoadConfig()
	claims := jwt.MapClaims{
		"userId": user.ID,
		"role":   user.Role,
		"exp":    time.Now().Add(AccessTokenTTL).Unix(),
		"iat":    time.Now().Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(cfg.JWTSecret))
}
