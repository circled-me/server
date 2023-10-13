package models

import (
	"server/utils"
	"time"
)

type AlbumShare struct {
	ID           uint64 `gorm:"primaryKey"`
	CreatedAt    int64
	UserID       uint64 `gorm:"not null"`
	User         User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumID      uint64 `gorm:"not null"`
	Album        Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token        string `gorm:"type:varchar(100);index:uniq_token,unique"`
	ExpiresAt    int64  `gorm:"not null"` // 0 indicates no expiration
	HideOriginal int    `gorm:"type:tinyint;not null"`
}

func NewAlbumShare(userID uint64, album uint64, expires int64, hideOriginal int) AlbumShare {
	expiresAt := int64(0)
	if expires > 0 {
		expiresAt = time.Now().Unix() + expires
	}
	return AlbumShare{
		UserID:       userID,
		AlbumID:      album,
		Token:        utils.Rand16BytesToBase62(),
		ExpiresAt:    expiresAt,
		HideOriginal: hideOriginal,
	}
}
