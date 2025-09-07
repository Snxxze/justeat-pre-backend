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
	
	EasySlipAPIKey  string
}

func LoadConfig() *Config {
	_ = godotenv.Load(".env",".env.local") //มี API KEY ของ EasySlip

	// helper ดึงค่าตัวแรกที่ไม่ว่าง
    firstNonEmpty := func(keys ...string) string {
        for _, k := range keys {
            if v, ok := os.LookupEnv(k); ok && v != "" {
				log.Printf("Found env var %s with length: %d", k, len(v))
                return v
            }
        }
		log.Printf("No env var found for keys: %v", keys)
        return ""
    }

	return &Config{
		DBDriver:  getEnv("DB_DRIVER", "sqlite"),
		DBSource:  getEnv("DB_SOURCE", "test.db"),
		Port:      getEnv("PORT", "8000"),
		JWTSecret: getEnv("JWT_SECRET", "changeme"),
		JWTTTL: 	 time.Duration(24) * time.Hour,	
		EasySlipAPIKey: firstNonEmpty("EASYSLIP_API_KEY", "EASYSLIP_TOKEN", "EASYSLIP_KEY"),
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
