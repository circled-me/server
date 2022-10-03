package models

type Like struct {
	CreatedAt   int
	UserID      uint64    `gorm:"primaryKey"`
	User        User      `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupPostID uint64    `gorm:"primaryKey"`
	GroupPost   GroupPost `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
