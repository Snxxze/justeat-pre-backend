package main

import (
	"fmt"
	"log"

	"backend/configs"
	"backend/middlewares"
	"backend/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := configs.LoadConfig()

	log.Printf("[MAIN] EasySlip API Key present=%v len=%d", cfg.EasySlipAPIKey != "", len(cfg.EasySlipAPIKey))
    
    if cfg.EasySlipAPIKey == "" {
        log.Fatal("EASYSLIP_API_KEY is required but not found in environment")
    }

	// DB
	configs.ConnectionDB()
	db := configs.DB()

	// migrate
	configs.SetupDatabase()

	if err := configs.SeedAdmin(); err != nil {
		log.Fatalf("seed admin failed: %v", err)
	}
	if err := configs.SeedLookups(); err != nil {
		log.Fatalf("seed lookups failed: %v", err)
	}

	// HTTP
	r := gin.Default()
	r.Use(middlewares.CORSMiddleware())
	routes.RegisterRoutes(r, db, cfg)

	port := configs.LoadConfig().Port
	addr := fmt.Sprintf(":%s", port)
	log.Println("🚀 Server running at", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
