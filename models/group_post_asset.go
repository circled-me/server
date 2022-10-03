package models

type GroupPostAsset struct {
	CreatedAt   int64
	GroupPostID uint64    `gorm:"primaryKey"`
	GroupPost   GroupPost `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID     uint64    `gorm:"primaryKey"`
	Asset       Asset     `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
