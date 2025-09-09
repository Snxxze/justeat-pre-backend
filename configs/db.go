package configs

import (
	"log"

	"backend/entity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func DB() *gorm.DB { return db }

func ConnectionDB() {
	cfg := LoadConfig()

	var (
		database *gorm.DB
		err      error
	)

	switch cfg.DBDriver {
	case "sqlite":
		database, err = gorm.Open(sqlite.Open(cfg.DBSource), &gorm.Config{})
	default:
		log.Fatalf("unsupported DB driver: %s", cfg.DBDriver)
	}

	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	db = database
}

func SetupDatabase() {
	// AutoMigrate ทั้งหมดของคุณ (มี entity อยู่แล้ว)
	if err := db.AutoMigrate(
		&entity.User{}, &entity.Admin{},
		&entity.RestaurantCategory{}, &entity.RestaurantStatus{}, &entity.Restaurant{},
		&entity.MenuType{}, &entity.MenuStatus{}, &entity.Menu{},
		&entity.OrderStatus{}, &entity.Order{}, &entity.OrderItem{},
		&entity.Cart{}, &entity.CartItem{},
		&entity.PaymentMethod{}, &entity.PaymentStatus{}, &entity.Payment{},
		&entity.RiderStatus{}, &entity.Rider{}, &entity.RiderWork{},
		&entity.ChatRoom{}, &entity.MessageType{}, &entity.Message{},
		&entity.PromoType{}, &entity.Promotion{}, &entity.UserPromotion{},
		&entity.Review{},
		&entity.IssueType{}, &entity.Report{},
		&entity.RestaurantApplication{},&entity.RiderApplication{},
	); err != nil {
		log.Fatalf("auto-migrate failed: %v", err)
	}
}
