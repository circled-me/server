package models

type GroupMessageReaction struct {
	ID       uint64 `gorm:"primaryKey;not null" json:"id"`
	UserID   uint64 `gorm:"primaryKey;not null" json:"user_id"`
	GroupID  uint64 `gorm:"-" json:"group_id"`
	Reaction string `gorm:"type:varchar(20);not null" json:"reaction"`
}
