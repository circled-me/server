package push

import (
	"log"
	"server/db"
	"server/models"
	"strconv"
)

func AlbumNewAssets(count int, albumId uint64, addeByUser *models.User) {
	// TODO: Use raw queries instead?
	album := models.Album{ID: albumId}
	if db.Instance.
		Preload("User").
		Preload("Contributors").
		First(&album).Error != nil {

		log.Print("Cannot find album?")
		return
	}
	what := strconv.Itoa(count) + " new photo"
	if count > 1 {
		what += "s"
	}
	notification := Notification{
		Title: "Album \"" + album.Name + "\"",
		Body:  addeByUser.Name + " added " + what + " to the album",
	}
	if album.UserID != addeByUser.ID {
		notification.UserToken = album.User.PushToken
		Send(&notification)
	}
	for _, c := range album.Contributors {
		if addeByUser.ID == c.UserID {
			continue
		}
		if c.User.ID != c.UserID {
			c.User.ID = c.UserID
			db.Instance.First(&c.User)
		}
		if c.User.PushToken == "" {
			continue
		}
		notification.UserToken = c.User.PushToken
		Send(&notification)
	}
}
