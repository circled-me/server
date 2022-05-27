package models

type Post struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int
	UpdatedAt int
	UserID    uint64
	User      User `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupID   uint64
	Group     Group
	Content   string `gorm:"type:text"`
}
