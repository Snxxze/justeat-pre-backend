package entity

import (
	"gorm.io/gorm"
)

type Option struct {
    gorm.Model
    Name       string `json:"name"`
    Type       string `json:"type"`
    MinSelect  int    `json:"minSelect"`
    MaxSelect  int    `json:"maxSelect"`
    IsRequired bool   `json:"isRequired"`
    SortOrder  int    `json:"sortOrder"`

    OptionValues []OptionValue `json:"optionValues"`

    Menus []Menu `gorm:"many2many:menu_options;" json:"-"`
}

