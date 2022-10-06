package models

type GroupUser struct {
	CreatedAt   int64  `gorm:"index"`
	GroupID     uint64 `gorm:"primaryKey"`
	Group       Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID      uint64 `gorm:"primaryKey"`
	User        User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CanInvite   bool
	IsAdmin     bool
	IsFavourite bool
	Colour      string `gorm:"type:varchar(10);not null"`
}
