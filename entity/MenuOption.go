package entity

type MenuOption struct {
	MenuID   uint `gorm:"primaryKey"`
  OptionID uint `gorm:"primaryKey"`
  SortOrder int `gorm:"not null;default:0"` 
}