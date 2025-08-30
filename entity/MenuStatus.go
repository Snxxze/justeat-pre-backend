package entity

import (
	"gorm.io/gorm"
)

type MenuStatus struct {
	gorm.Model
	StatusName string `json:"statusName"`

	// ไม่จำเป็นต้องส่ง relation เสมอ
	Menus []Menu `json:"-"`
}
