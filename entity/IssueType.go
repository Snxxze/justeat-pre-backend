package entity

import (
	"gorm.io/gorm"
)

type IssueType struct {
	gorm.Model
	TypeName string  
	Reports  []Report
}