package configs

import (
	"log"

	"backend/entity"
	"golang.org/x/crypto/bcrypt"
)

// สร้าง admin ครั้งแรก
func SeedAdmin() error {
	db := DB()
	email := getEnv("ADMIN_EMAIL", "")
	pass := getEnv("ADMIN_PASSWORD", "")
	if email == "" || pass == "" {
		log.Println("⚠️ skip seeding admin: missing ADMIN_EMAIL/ADMIN_PASSWORD")
		return nil
	}

	var count int64
	db.Model(&entity.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		log.Println("ℹ️ admin already exists:", email)
		return nil
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	admin := entity.User{
		Email:     email,
		Password:  string(hash),
		FirstName: "Admin",
		LastName:  "Seed",
		Role:      "admin",
	}
	return db.Create(&admin).Error
}

// Seed ค่า lookup/status เริ่มต้น
func SeedLookups() error {
	db := DB()

	// Mock Users
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	db.FirstOrCreate(&entity.User{}, entity.User{
			Email:       "customer@example.com",
			Password:    string(hash),
			FirstName:   "Cus",
			LastName:    "Tomer",
			Role:        "customer",
	})
	db.FirstOrCreate(&entity.User{}, entity.User{
			Email:       "owner@example.com",
			Password:    string(hash),
			FirstName:   "Own",
			LastName:    "Er",
			Role:        "owner",
	})
	db.FirstOrCreate(&entity.User{}, entity.User{
			Email:       "rider@example.com",
			Password:    string(hash),
			FirstName:   "R",
			LastName:    "Ider",
			Role:        "rider",
	})

	
	// Restaurant
	db.FirstOrCreate(&entity.Restaurant{}, entity.Restaurant{
    Name:        "Pizza Town",
    Address:     "Bangkok",
    Description: "Best pizza in town",
    RestaurantCategoryID: 1, // Cafe
    RestaurantStatusID:   1, // Open
    UserID: 2, // owner@example.com
	})
	db.FirstOrCreate(&entity.Menu{}, entity.Menu{
    MenuName: "Cappuccino",
    Detail:   "Hot coffee with milk foam",
    Price:    50,
    RestaurantID: 1,
    MenuTypeID:   1, // Drink
    MenuStatusID: 1, // Available
	})
	db.FirstOrCreate(&entity.Menu{}, entity.Menu{
			MenuName: "Margherita Pizza",
			Detail:   "Cheese & Tomato",
			Price:    199,
			RestaurantID: 1,
			MenuTypeID:   2, // Main Dish
			MenuStatusID: 1,
	})
	// RestaurantStatus
	db.FirstOrCreate(&entity.RestaurantStatus{}, entity.RestaurantStatus{StatusName: "Open"})
	db.FirstOrCreate(&entity.RestaurantStatus{}, entity.RestaurantStatus{StatusName: "Closed"})

	// RestaurantCate
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Rics Dishes"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Noodles"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Coffee & Tea"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Fast Food"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Healthy"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Bubble Tea"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Bakery"})
	
	// Menu
	db.FirstOrCreate(&entity.MenuStatus{}, entity.MenuStatus{StatusName: "Available"})
	db.FirstOrCreate(&entity.MenuStatus{}, entity.MenuStatus{StatusName: "Out of Stock"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "Drink"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "Main Dish"})

	// Order Status
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Paid"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Delivering"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Completed"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Cancelled"})

	// Payment Method
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "PromptPay"})
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "Credit Card"})
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "Cash on Delivery"})

	// Payment Status
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Paid"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Failed"})

	// Rider
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Available"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Delivering"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Offline"})

	// Message Type
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "Text"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "Image"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "System"})

	// Promotion Type
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Discount"})
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Percent"})
	// db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Free Delivery"})

	// Issue / Report
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Wrong Item"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Delivery Late"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "System Failed"})

	log.Println("✅ Lookup tables seeded")
	return nil
}
