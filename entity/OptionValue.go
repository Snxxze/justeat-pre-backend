package entity

import (
	"gorm.io/gorm"
)

type OptionValue struct {
	gorm.Model
	OptionID        uint
	Option          Option 
	ValueName       string 
	PriceAdjustment int64 
	DefaultSelect   bool   
	IsAvailable     bool   
	SortOrder       int   
}