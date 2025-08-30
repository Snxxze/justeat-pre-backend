package main

import (
	"fmt"
	"log"

	"backend/configs"
	"backend/entity"
	"backend/routes"

	"github.com/gin-gonic/gin"
)

func main() {
	// DB
	configs.ConnectionDB()
	db := configs.DB()

	// join table (many2many Menu<->Option)
	if err := db.SetupJoinTable(&entity.Menu{}, "Options", &entity.MenuOption{}); err != nil {
		log.Fatalf("setup join table failed: %v", err)
	}

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
	routes.RegisterRoutes(r)

	port := configs.LoadConfig().Port
	addr := fmt.Sprintf(":%s", port)
	log.Println("🚀 Server running at", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
