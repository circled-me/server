package push

import (
	"log"
	"server/config"
	"server/db"
	"server/models"
	"strconv"
)

func AlbumNewContributor(newUser, albumId uint64, mode int, addeByUser *models.User) {
	if config.PUSH_SERVER == "" {
		return
	}
	receiver := models.User{ID: newUser}
	if err := db.Instance.First(&receiver).Error; err != nil {
		log.Printf("AlbumNewContributor DB error: %v", err)
		return
	}
	album := models.Album{ID: albumId}
	if db.Instance.First(&album).Error != nil {
		log.Print("Cannot find album?")
		return
	}
	what := "a viewer"
	if mode == models.ContributorCanEdit {
		what = "an editor"
	}
	notification := Notification{
		UserToken: receiver.PushToken,
		Title:     "Album \"" + album.Name + "\"",
		Body:      addeByUser.Name + " added you as " + what,
		Data: map[string]string{
			"type":  NotificationTypeNewAssetsInAlbum,
			"album": strconv.Itoa(int(albumId)),
		},
	}
	Send(&notification)
}

func AlbumNewAssets(count int, albumId uint64, addeByUser *models.User) {
	if config.PUSH_SERVER == "" {
		return
	}
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
		Body:  addeByUser.Name + " added " + what,
		Data: map[string]string{
			"type":  NotificationTypeNewAssetsInAlbum,
			"album": strconv.Itoa(int(albumId)),
		},
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
