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

	// Restaurant
	db.FirstOrCreate(&entity.RestaurantStatus{}, entity.RestaurantStatus{StatusName: "Open"})
	db.FirstOrCreate(&entity.RestaurantStatus{}, entity.RestaurantStatus{StatusName: "Closed"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Cafe"})
	db.FirstOrCreate(&entity.RestaurantCategory{}, entity.RestaurantCategory{CategoryName: "Fast Food"})

	// Menu
	db.FirstOrCreate(&entity.MenuStatus{}, entity.MenuStatus{StatusName: "Available"})
	db.FirstOrCreate(&entity.MenuStatus{}, entity.MenuStatus{StatusName: "Out of Stock"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "Drink"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "Main Dish"})

	// Order
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Paid"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Delivering"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Completed"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Cancelled"})

	// Payment
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "PromptPay"})
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "Credit Card"})
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "Cash on Delivery"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Paid"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Failed"})

	// Rider
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Available"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Delivering"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "Offline"})

	// Message
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "Text"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "Image"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{TypeName: "System"})

	// Promotion
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Discount"})
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Free Delivery"})
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Percent"})

	// Issue / Report
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Delivery Late"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Wrong Item"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Payment Failed"})

	log.Println("✅ Lookup tables seeded")
	return nil
}
