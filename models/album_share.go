package models

import "server/utils"

type AlbumShare struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int64
	UserID    uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:1"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumID   uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:2"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token     string `gorm:"type:varchar(100);index:uniq_token,unique"`
}

func NewAlbumShare(userID uint64, album uint64) AlbumShare {
	return AlbumShare{
		UserID:  userID,
		AlbumID: album,
		Token:   utils.Rand16BytesToBase62(),
	}
}
