package models

type GroupPost struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int64  `gorm:"index:group_order,priority:2"`
	GroupID   uint64 `gorm:"index:group_order,priority:1"`
	Group     Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Content   string `gorm:"type:text"`
}
