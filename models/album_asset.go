package models

type AlbumAsset struct {
	CreatedAt int64  `gorm:"index:album_order,priority:2"`
	AlbumID   uint64 `gorm:"primaryKey;index:album_order,priority:1"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID   uint64 `gorm:"primaryKey"`
	Asset     Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
