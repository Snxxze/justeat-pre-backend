package configs

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DBDriver  string
	DBSource  string
	Port      string
	JWTSecret string
	JWTTTL		time.Duration
	EasySlipAPIKey string `mapstructure:"EASYSLIP_API_KEY"`
}

func LoadConfig() *Config {
	_ = godotenv.Load() // ถ้าไม่มี .env ก็ข้าม

	return &Config{
		DBDriver:  getEnv("DB_DRIVER", "sqlite"),
		DBSource:  getEnv("DB_SOURCE", "test.db"),
		Port:      getEnv("PORT", "8000"),
		JWTSecret: getEnv("JWT_SECRET", "changeme"),
		JWTTTL: 	 time.Duration(24) * time.Hour,	
	}
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// Helper เผื่อไฟล์อื่นต้องใช้ (เช่น seed)
func MustGetEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		log.Fatalf("missing env: %s", key)
	}
	return v
}
