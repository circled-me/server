package models

type Group struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   int
	UpdatedAt   int
	CreatedByID uint64
	CreatedBy   User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name        string `gorm:"type:varchar(300)"`
}
