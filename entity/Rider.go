package entity

import "gorm.io/gorm"

type Rider struct {
    gorm.Model
    VehiclePlate string `json:"vehiclePlate" gorm:"column:vehicle_plate;type:varchar(50);not null"`
    License      string `json:"license"      gorm:"column:license;type:varchar(50);not null"` // เลขใบขับขี่
    NationalID   string `json:"nationalId"   gorm:"column:national_id;type:varchar(20);not null"`
    Zone         string `json:"zone"         gorm:"column:zone;type:varchar(100);not null"`

    DriveCard    string `json:"driveCard,omitempty" gorm:"column:drive_card;type:longtext"`

    RiderStatusID uint        `json:"riderStatusId"`
    RiderStatus   RiderStatus `json:"-"`

    AdminID *uint  `json:"adminId,omitempty"`
    Admin   *Admin `json:"-"`

    UserID uint `json:"userId" gorm:"uniqueIndex"` // หนึ่ง user มี rider เดียว
    User   User `json:"-"`
}