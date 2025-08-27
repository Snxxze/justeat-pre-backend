// main.go
package main

import (
	"log"

	"backend/configs"
	"backend/entity"
)

func main() {
	// 1) เปิด/เชื่อมต่อ DB (สร้างไฟล์ test.db อัตโนมัติถ้ายังไม่มี)
	configs.ConnectionDB()
	db := configs.DB()

	// 2) ตั้งค่า join table สำหรับ many2many Menu<->Option (มี field เพิ่ม เช่น sort_order)
	if err := db.SetupJoinTable(&entity.Menu{}, "Options", &entity.MenuOption{}); err != nil {
		log.Fatalf("setup join table failed: %v", err)
	}

	// 3) AutoMigrate ตามที่เขียนไว้ใน configs.SetupDatabase()
	configs.SetupDatabase()

	log.Println("SQLite initialized & migrated: test.db ✅")
}
