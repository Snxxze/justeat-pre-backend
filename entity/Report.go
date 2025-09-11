package entity

import (
	"time"
	"gorm.io/gorm"
)

type Report struct {
	gorm.Model
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	PhoneNumber string     `json:"phoneNumber"`
	Description string     `json:"description"`
	DateAt      *time.Time `json:"dateAt,omitempty"`
	Picture     string     `json:"picture"`

	IssueTypeID uint      `json:"issueTypeId"`
	IssueType   IssueType `json:"-"` // preload เฉพาะตอน detail

	UserID uint `json:"userId"`
	User   User `json:"-"` // preload เฉพาะตอนต้องการข้อมูล user

	AdminID uint  `json:"adminId"`
	Admin   Admin `json:"-"` // preload เฉพาะตอนต้องการข้อมูล admin


	
	// ✅ เพิ่มสถานะ
	Status string `json:"status" gorm:"type:varchar(50);default:'pending'"`
}
