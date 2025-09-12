package entity

import (
	"gorm.io/gorm"
)

type Restaurant struct {
	gorm.Model
	Name        string `json:"name"`
	Address     string `json:"address"`
	Description string `json:"description"`
	Picture     string `json:"pictureBase64,omitempty" gorm:"column:picture_base64;type:longtext"`

	OpeningTime string `json:"openingTime"`
	ClosingTime string `json:"closingTime"`

	RestaurantCategoryID uint               `json:"restaurantCategoryId"`
	RestaurantCategory   RestaurantCategory `json:"-"` // preload เฉพาะตอนต้องการ

	RestaurantStatusID uint             `json:"restaurantStatusId"`
	RestaurantStatus   RestaurantStatus `json:"-"` // preload เฉพาะตอน detail

	PromptPay string `json:"promptPay" gorm:"column:prompt_pay;type:varchar(32)"` // promptPay จะเป็นเบอร์ 10 หลัก หรือเลขบัตรประชาชน 13 หลัก

	UserID uint `json:"userId"` // owner
	User   User `json:"-"` // preload เฉพาะตอนต้องการข้อมูลเจ้าของร้าน

	AdminID *uint  `json:"adminId,omitempty"`
	Admin   *Admin `json:"-"` // preload เฉพาะตอนที่ admin ต้องการจัดการ

	Menus   []Menu   `json:"-"` // preload แค่ endpoint /restaurants/:id/menus
	Orders  []Order  `json:"-"` // preload แค่ endpoint /restaurants/:id/orders
	Reviews []Review `json:"-"` // preload แค่ endpoint /restaurants/:id/reviews
}

