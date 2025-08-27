package entity

import (
	"gorm.io/gorm"
)

type MenuType struct {
	gorm.Model
	TypeName string 
	Menus    []Menu
}