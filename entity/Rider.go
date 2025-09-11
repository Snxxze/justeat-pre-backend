package entity

import (
	"gorm.io/gorm"
)

type Rider struct {
	gorm.Model
	VehiclePlate string `json:"vehiclePlate"`
	License      string `json:"license"`
	
	DriveCard    string `json:"driveCard,omitempty" gorm:"column:drive_card;type:longtext"`

	RiderStatusID uint        `json:"riderStatusId"`
	RiderStatus   RiderStatus `json:"-"` // preload เฉพาะตอน detail

	AdminID *uint  `json:"adminId,omitempty"`
	Admin   *Admin `json:"-"` // preload เฉพาะตอนที่ admin ต้องการจัดการ

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload เฉพาะเวลาต้องการชื่อ/ข้อมูล user

	Works []RiderWork `json:"-"` // preload เฉพาะ endpoint ประวัติการทำงาน
}

