package models

type Comment struct {
	ID              uint64 `gorm:"primaryKey"`
	CreatedAt       int
	UserID          uint64
	User            User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PostID          uint64
	Post            Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ParentCommentID uint64
	ParentComment   *Comment `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AssetID         uint64
	Asset           Asset  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Content         string `gorm:"type:text"`
}
