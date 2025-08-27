package entity

import (
	"time"
	"gorm.io/gorm"
)

type Report struct {
	gorm.Model
	Name        string 
	Email       string 
	PhoneNumber string 
	Description string
	DateAt      *time.Time
	Picture     string 

	IssueTypeID uint
	IssueType   IssueType 
	UserID      uint
	User        User 
	AdminID     uint
	Admin       Admin 
}