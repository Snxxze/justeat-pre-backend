package configs

import (
	"backend/entity"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func DB() *gorm.DB {
	return db
}

func ConnectionDB() {
	database, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}
	db = database
}

func SetupDatabase() {

	// Migrate the schema
	db.AutoMigrate(
		&entity.User{}, &entity.Admin{},
		&entity.RestaurantCategory{}, &entity.RestaurantStatus{}, &entity.Restaurant{},
		&entity.MenuType{}, &entity.MenuStatus{}, &entity.Menu{}, &entity.MenuOption{},
		&entity.Option{}, &entity.OptionValue{},
		&entity.OrderStatus{}, &entity.Order{}, &entity.OrderItem{}, &entity.OrderItemSelection{},
		&entity.PaymentMethod{}, &entity.PaymentStatus{}, &entity.Payment{},
		&entity.RiderStatus{}, &entity.Rider{}, &entity.RiderWork{},
		&entity.ChatRoom{}, &entity.MessageType{}, &entity.Message{},
		&entity.PromoType{}, &entity.Promotion{}, &entity.UserPromotion{},
		&entity.Review{},
		&entity.IssueType{}, &entity.Report{},
	)
}