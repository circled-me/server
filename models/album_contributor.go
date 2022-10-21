package models

type AlbumContributor struct {
	CreatedAt int64
	UserID    uint64 `gorm:"primaryKey"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumID   uint64 `gorm:"primaryKey"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
