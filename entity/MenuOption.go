package entity

type MenuOption struct {
	MenuID    uint `gorm:"primaryKey;index:idx_menu_option"`
	OptionID  uint `gorm:"primaryKey;index:idx_menu_option"`
	// SortOrder int  `gorm:"not null;default:0"`
}