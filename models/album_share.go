package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

type AlbumShare struct {
	ID        uint64 `gorm:"primaryKey"`
	CreatedAt int
	UserID    uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:1"`
	User      User   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	AlbumID   uint64 `gorm:"not null;index:uniq_user_album_share,unique,priority:2"`
	Album     Album  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Token     string `gorm:"type:varchar(100);index:uniq_token,unique"`
}

func rand16BytesToBase62() string {
	buf := make([]byte, 16)
	_, err := rand.Read(buf)
	if err != nil {
		fmt.Println("error:", err)
		panic(err)
	}
	var i big.Int
	return i.SetBytes(buf).Text(62)
}

func NewAlbumShare(user uint64, album uint64) AlbumShare {
	return AlbumShare{
		UserID:  user,
		AlbumID: album,
		Token:   rand16BytesToBase62() + rand16BytesToBase62(),
	}
}
