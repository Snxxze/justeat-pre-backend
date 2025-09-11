package entity

type MenuOption struct {
	MenuID   uint `gorm:"primaryKey" json:"menuId"`
	OptionID uint `gorm:"primaryKey" json:"optionId"`
	SortOrder int `gorm:"not null;default:0" json:"sortOrder"`
}
