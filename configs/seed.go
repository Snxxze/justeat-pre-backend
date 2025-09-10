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
		Email:     "customer@example.com",
		Password:  string(hash),
		FirstName: "Cus",
		LastName:  "Tomer",
		Role:      "customer",
	})
	db.FirstOrCreate(&entity.User{}, entity.User{
		Email:     "owner@example.com",
		Password:  string(hash),
		FirstName: "Own",
		LastName:  "Er",
		Role:      "owner",
	})
	db.FirstOrCreate(&entity.User{}, entity.User{
		Email:     "rider@example.com",
		Password:  string(hash),
		FirstName: "R",
		LastName:  "Ider",
		Role:      "rider",
	})

	// Restaurant
	db.FirstOrCreate(&entity.Restaurant{}, entity.Restaurant{
		Name:                 "Pizza Town",
		Address:              "Bangkok",
		Description:          "Best pizza in town",
		RestaurantCategoryID: 1, // Cafe
		RestaurantStatusID:   1, // Open
		UserID:               2, // owner@example.com
	})
	db.FirstOrCreate(&entity.Menu{}, entity.Menu{
		Name:         "Cappuccino",
		Detail:       "Hot coffee with milk foam",
		Price:        50,
		RestaurantID: 1,
		MenuTypeID:   1, // Drink
		MenuStatusID: 1, // Available
	})
	db.FirstOrCreate(&entity.Menu{}, entity.Menu{
		Name:         "Margherita Pizza",
		Detail:       "Cheese & Tomato",
		Price:        199,
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

	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "เมนูหลัก"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "ของทานเล่น"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "ของหวาน"})
	db.FirstOrCreate(&entity.MenuType{}, entity.MenuType{TypeName: "เครื่องดื่ม"})

	// Order Status
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Preparing"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Delivering"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Completed"})
	db.FirstOrCreate(&entity.OrderStatus{}, entity.OrderStatus{StatusName: "Cancelled"})

	// Payment Method
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "PromptPay"})
	db.FirstOrCreate(&entity.PaymentMethod{}, entity.PaymentMethod{MethodName: "Cash on Delivery"})

	// Payment Status
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Pending"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Paid"})
	db.FirstOrCreate(&entity.PaymentStatus{}, entity.PaymentStatus{StatusName: "Failed"})

	// Rider
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "OFFLINE"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "ONLINE"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "ASSIGNED"})
	db.FirstOrCreate(&entity.RiderStatus{}, entity.RiderStatus{StatusName: "COMPLETED"})

	// Message Type
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{Name: "TEXT"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{Name: "IMAGE"})
	db.FirstOrCreate(&entity.MessageType{}, entity.MessageType{Name: "SYSTEM"})

	// Promotion Type
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Discount"})
	db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Percent"})
	// db.FirstOrCreate(&entity.PromoType{}, entity.PromoType{NameType: "Free Delivery"})

	// Issue / Report
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Wrong Item"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "Delivery Late"})
	db.FirstOrCreate(&entity.IssueType{}, entity.IssueType{TypeName: "System Failed"})

	// -------------------- Owners (owner1..owner4) --------------------
	{
		// ถ้ายังไม่มี owners เหล่านี้ จะสร้างให้ด้วยรหัสผ่าน 123456
		// (ใช้ hash ตัวเดียวกับด้านบน)
		type ownerSeed struct {
			Email, First, Last string
		}
		owners := []ownerSeed{
			{"owner1@example.com", "Owner", "One"},
			{"owner2@example.com", "Owner", "Two"},
			{"owner3@example.com", "Owner", "Three"},
			{"owner4@example.com", "Owner", "Four"},
		}
		for _, o := range owners {
			db.Where("email = ?", o.Email).
				Attrs(entity.User{
					Email:     o.Email,
					Password:  string(hash),
					FirstName: o.First,
					LastName:  o.Last,
					Role:      "owner",
				}).
				FirstOrCreate(&entity.User{})
		}
	}

	// -------------------- Resolve IDs we need --------------------
	var stOpen entity.RestaurantStatus
	db.First(&stOpen, "status_name = ?", "Open")

	var mtMain, mtSide, mtDrink, mtDessert entity.MenuType
	db.First(&mtMain, "type_name = ?", "เมนูหลัก")
	db.First(&mtSide, "type_name = ?", "ของทานเล่น")
	db.First(&mtDrink, "type_name = ?", "เครื่องดื่ม")
	db.First(&mtDessert, "type_name = ?", "ของหวาน")

	var msAvail entity.MenuStatus
	db.First(&msAvail, "status_name = ?", "Available")

	// หมวดหมู่จากที่คุณ seed ไว้
	var catFastFood, catCoffeeTea, catNoodles, catHealthy, catBakery, catBubbleTea entity.RestaurantCategory
	db.First(&catFastFood, "category_name = ?", "Fast Food")
	db.First(&catCoffeeTea, "category_name = ?", "Coffee & Tea")
	db.First(&catNoodles, "category_name = ?", "Noodles")
	db.First(&catHealthy, "category_name = ?", "Healthy")
	db.First(&catBakery, "category_name = ?", "Bakery")
	db.First(&catBubbleTea, "category_name = ?", "Bubble Tea")

	// helper: ดึง userID จาก email (panic ไม่ได้—ถ้าอยากกันพังให้เช็ค error เอง)
	getUserID := func(email string) uint {
		var u entity.User
		db.First(&u, "email = ?", email)
		return u.ID
	}

	// helper: create/upsert ร้าน โดยให้ unique ที่ Name (ถ้าคุณอยากกันชนชื่อซ้ำข้าม owner ให้เพิ่มคีย์อื่น)
	createRestaurant := func(name, addr, desc string, catID, ownerID uint) entity.Restaurant {
		r := entity.Restaurant{
			Name:                 name,
			Address:              addr,
			Description:          desc,
			OpeningTime:          "09:00",
			ClosingTime:          "21:00",
			RestaurantCategoryID: catID,
			RestaurantStatusID:   stOpen.ID,
			UserID:               ownerID,
		}
		db.Where("name = ?", name).Attrs(r).FirstOrCreate(&r) // idempotent โดยยึดชื่อร้านเป็นตัวคุม
		return r
	}

	// helper: create/upsert เมนู โดย unique (Name + RestaurantID)
	createMenu := func(rest entity.Restaurant, name, detail string, price int64, mt entity.MenuType) {
		m := entity.Menu{
			Name:         name,
			Detail:       detail,
			Price:        price,
			RestaurantID: rest.ID,
			MenuTypeID:   mt.ID,
			MenuStatusID: msAvail.ID,
			// Image: "", // ถ้าจะใส่ base64/URL ค่อยเติมได้
		}
		db.Where("name = ? AND restaurant_id = ?", name, rest.ID).Attrs(m).FirstOrCreate(&m)
	}

	// -------------------- ร้าน + เจ้าของที่กำหนดชัดเจน --------------------
	// 1) Pizza Town -> owner1
	pizzaTown := createRestaurant(
		"Pizza Town", "Bangkok", "Best pizza in town",
		catFastFood.ID, getUserID("owner1@example.com"),
	)
	// เมนู 5 รายการ (ตัวอย่าง)
	createMenu(pizzaTown, "Margherita", "ชีส+ซอสมะเขือเทศ", 199, mtMain)
	createMenu(pizzaTown, "Pepperoni", "เปปเปอโรนีเต็มแผ่น", 229, mtMain)
	createMenu(pizzaTown, "Hawaiian", "สับปะรด แฮม ฉ่ำ", 219, mtMain)
	createMenu(pizzaTown, "BBQ Chicken", "ซอสบาร์บีคิวไก่", 239, mtMain)
	createMenu(pizzaTown, "Garlic Bread", "ขนมปังกระเทียมหอมเนย", 69, mtSide)

	// 2) Noodle House -> owner2
	noodle := createRestaurant(
		"Noodle House", "Bangkok", "เส้นสด น้ำซุปกลมกล่อม",
		catNoodles.ID, getUserID("owner2@example.com"),
	)
	createMenu(noodle, "เส้นเล็กน้ำใส", "กลิ่นหอมกระเทียมเจียว", 55, mtMain)
	createMenu(noodle, "เส้นใหญ่ต้มยำ", "เข้มข้น เปรี้ยว เผ็ด", 65, mtMain)
	createMenu(noodle, "บะหมี่แห้งหมูแดง", "หมูแดงโฮมเมด", 60, mtMain)
	createMenu(noodle, "เกาเหลา", "ไม่เอาเส้น เน้นเครื่อง", 60, mtMain)
	createMenu(noodle, "ชาดำเย็น", "หวานเย็นชื่นใจ", 25, mtDrink)

	// 3) Healthy Garden -> owner3
	healthy := createRestaurant(
		"Healthy Garden", "Bangkok", "สลัดและอาหารคลีน",
		catHealthy.ID, getUserID("owner3@example.com"),
	)
	createMenu(healthy, "สลัดอกไก่", "ผักสด อกไก่ย่าง", 85, mtMain)
	createMenu(healthy, "สลัดซีซาร์", "น้ำสลัดโฮมเมด", 89, mtMain)
	createMenu(healthy, "ข้าวกล้องอกไก่", "โปรตีนสูง ไขมันต่ำ", 79, mtMain)
	createMenu(healthy, "ควินัวโบว์ล", "ธัญพืชครบถ้วน", 119, mtMain)
	createMenu(healthy, "สมูทตี้ผักโขม", "น้ำตาลต่ำ", 69, mtDrink)

	// 4) Burger Street -> owner4
	burger := createRestaurant(
		"Burger Street", "Bangkok", "เบอร์เกอร์โฮมเมด",
		catFastFood.ID, getUserID("owner4@example.com"),
	)
	createMenu(burger, "ชีสเบอร์เกอร์", "เนื้อฉ่ำ ชีสเยิ้ม", 109, mtMain)
	createMenu(burger, "ดับเบิลชีสเบอร์เกอร์", "อิ่มจัดเต็ม", 149, mtMain)
	createMenu(burger, "ฟรายส์", "กรอบนอกนุ่มใน", 49, mtSide)
	createMenu(burger, "นักเก็ต", "ไก่คุณภาพ", 59, mtSide)
	createMenu(burger, "โซดามะนาว", "ซ่าและสดชื่น", 39, mtDrink)

	// 5) Sweet Bakery -> owner1 (เพิ่มอีกร้านให้ owner1)
	bakery := createRestaurant(
		"Sweet Bakery", "Bangkok", "เบเกอรี่หอมกรุ่นจากเตา",
		catBakery.ID, getUserID("owner1@example.com"),
	)
	createMenu(bakery, "ครัวซองต์เนยสด", "อบใหม่ทุกเช้า", 55, mtDessert)
	createMenu(bakery, "ครัวซองต์ช็อกโกแลต", "เข้มข้น", 65, mtDessert)
	createMenu(bakery, "ชีสเค้ก", "เนียนนุ่ม", 95, mtDessert)
	createMenu(bakery, "บานอฟฟี่", "กล้วย-คาราเมล-ครีม", 89, mtDessert)
	createMenu(bakery, "อเมริกาโน่ร้อน", "คั่วกลาง", 55, mtDrink)

	// 6) Boba Land -> owner2 (ถ้าต้องการครบ 5 ร้าน “เพิ่ม” จาก Pizza Town จะเป็น 5 ใหม่ + 1 เดิม)
	boba := createRestaurant(
		"Boba Land", "Bangkok", "ชานมไข่มุกและเครื่องดื่ม",
		catBubbleTea.ID, getUserID("owner2@example.com"),
	)
	createMenu(boba, "ชานมไข่มุก", "ไข่มุกหนึบ", 59, mtDrink)
	createMenu(boba, "ชาเขียวมะลิ", "หอมละมุน", 49, mtDrink)
	createMenu(boba, "นมสดบราวน์ชูการ์", "หวานมันกลมกล่อม", 69, mtDrink)
	createMenu(boba, "ช็อกโกแลตเย็น", "เข้มเต็มแก้ว", 59, mtDrink)
	createMenu(boba, "ผลไม้รวมโซดา", "สดชื่นซาบซ่า", 49, mtDrink)

	log.Println("Lookup tables seeded")
	return nil
}
