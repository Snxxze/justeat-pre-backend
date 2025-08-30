package entity

import (
	"gorm.io/gorm"
)

type Option struct {
	gorm.Model
	OptionName string `json:"optionName"`
	OptionType string `json:"optionType"`
	MinSelect  int    `json:"minSelect"`
	MaxSelect  int    `json:"maxSelect"`
	IsRequired bool   `json:"isRequired"`
	SortOrder  int    `json:"sortOrder"`

	// preload option values บ่อย → keep
	OptionValues []OptionValue `json:"optionValues"`

	// ไม่จำเป็นต้องส่ง relation กลับ
	Menus []Menu `gorm:"many2many:menu_options;" json:"-"`
}

