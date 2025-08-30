package entity

import (
	"gorm.io/gorm"
)

type IssueType struct {
	gorm.Model
	TypeName string `json:"typeName"`

	// ไม่ preload reports เสมอ เพราะจะ response หนัก
	Reports []Report `json:"-"`
}