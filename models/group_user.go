package models

type GroupUser struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int    `gorm:"index"`
	GroupID   uint64 `gorm:"index:uniq_g_u,priority:1,unique;index:idx_u_g,priority:2;"`
	Group     Group  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID    uint64 `gorm:"index:uniq_g_u,priority:2,unique;index:idx_u_g,priority:1;"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	CanInvite bool
	IsAdmin   bool
}
