package models

type AssetTag struct {
	CreatedAt int
	TagID     uint64 `gorm:"primaryKey;"`
	Tag       Tag    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID   uint64 `gorm:"primaryKey"`
	Asset     Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
