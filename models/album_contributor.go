package models

type AlbumContributor struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int64
	UserID    uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:1"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumID   uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:2"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}
