package models

type FavouriteAsset struct {
	UserID       uint64     `gorm:"primaryKey"`
	User         User       `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID      uint64     `gorm:"primaryKey"`
	Asset        Asset      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumAssetID *uint64    `gorm:"null;default null"`
	AlbumAsset   AlbumAsset `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}
