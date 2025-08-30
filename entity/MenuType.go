package entity

import (
	"gorm.io/gorm"
)

type MenuType struct {
	gorm.Model
	TypeName string `json:"typeName"`

	// ซ่อน relation เพื่อไม่ให้ response บวม
	Menus []Menu `json:"-"`
}
