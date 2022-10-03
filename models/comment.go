package models

type Comment struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   int64
	UserID      uint64
	User        User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupPostID uint64
	GroupPost   GroupPost `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID     uint64
	Asset       Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Content     string `gorm:"type:text"`
}
