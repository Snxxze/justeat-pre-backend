package entity

import (
	"gorm.io/gorm"
)

type Option struct {
	gorm.Model
	OptionName string 
	OptionType string 
	MinSelect  int   
	MaxSelect  int    
	IsRequired bool   
	SortOrder  int    

	OptionValues []OptionValue
	Menus []Menu `gorm:"many2many:menu_options;"`
}