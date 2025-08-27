package entity

import (
	"gorm.io/gorm"
)

type MenuStatus struct {
	gorm.Model
	StatusName string 
	Menus      []Menu
}