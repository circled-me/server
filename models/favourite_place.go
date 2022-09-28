package models

type FavouritePlace struct {
	PlaceID    uint64 `gorm:"primaryKey"`
	Place      Place  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID     uint64 `gorm:"primaryKey"`
	CustomName string `gorm:"type:varchar(100)"`
}
