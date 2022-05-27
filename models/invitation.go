package models

type Invitation struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int
	UserID    uint64
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token     string `gorm:"type:varchar(120);unique"`
}
