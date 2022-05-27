package models

type GroupPost struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int    `gorm:"index:group_order,priority:2"`
	GroupID   uint64 `gorm:"index:group_order,priority:1"`
	Group     Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	PostID    uint64
	Post      Post `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
