package models

type Like struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int
	UserID    uint64
	User      User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PostID    uint64
	Post      Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID   uint64
	Asset     Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
