package models

type GroupMessage struct {
	ID      uint64 `gorm:"primaryKey"`
	Stamp   uint64 `gorm:"index:group_order;unique,priority:2"`
	GroupID uint64 `gorm:"index:group_order;unique,priority:1"`
	Group   Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID  uint64 `gorm:"index:group_order;unique,priority:3"`
	User    User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Content string `gorm:"type:varchar(5000)"`
}
