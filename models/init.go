package models

import (
	"log"
	"math/rand"
	"server/db"
	"time"
)

func Init() {
	// Seed the random number generator - required for User.Salt
	rand.Seed(time.Now().UnixNano())

	es := []error{}
	es = append(es, db.Instance.AutoMigrate(&Album{}))
	es = append(es, db.Instance.AutoMigrate(&AlbumAsset{}))
	es = append(es, db.Instance.AutoMigrate(&AlbumContributor{}))
	es = append(es, db.Instance.AutoMigrate(&AlbumAsset{}))
	es = append(es, db.Instance.AutoMigrate(&AlbumShare{}))
	es = append(es, db.Instance.AutoMigrate(&Asset{}))
	es = append(es, db.Instance.AutoMigrate(&Face{}))
	es = append(es, db.Instance.AutoMigrate(&FavouriteAsset{}))
	es = append(es, db.Instance.AutoMigrate(&Grant{}))
	es = append(es, db.Instance.AutoMigrate(&Group{}))
	es = append(es, db.Instance.AutoMigrate(&GroupMessage{}))
	es = append(es, db.Instance.AutoMigrate(&GroupMessageReaction{}))
	es = append(es, db.Instance.AutoMigrate(&GroupUser{}))
	es = append(es, db.Instance.AutoMigrate(&Location{}))
	es = append(es, db.Instance.AutoMigrate(&Place{}))
	es = append(es, db.Instance.AutoMigrate(&Person{}))
	es = append(es, db.Instance.AutoMigrate(&UploadRequest{}))
	es = append(es, db.Instance.AutoMigrate(&User{}))
	es = append(es, db.Instance.AutoMigrate(&VideoCall{}))

	for _, e := range es {
		if e != nil {
			log.Printf("Auto-migrate error: %v", e)
		}
	}
}
