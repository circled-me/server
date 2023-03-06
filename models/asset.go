package models

import (
	"path/filepath"
	"server/db"
	"server/storage"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	AssetTypeOther = 0
	AssetTypeImage = 1
	AssetTypeVideo = 2

	presignViewURLFor      = time.Hour * 24 * 7
	presignValidAtLeastFor = time.Minute * 30
)

type Asset struct {
	ID                  uint64 `gorm:"primaryKey"`
	UserID              uint64 `gorm:"index:uniq_remote_id,unique,priority:1;not null;index:user_asset_created,priority:1"`
	RemoteID            string `gorm:"type:varchar(300);index:uniq_remote_id,unique,priority:2;not null"`
	CreatedAt           int64  `gorm:"index:user_asset_created,priority:3"`
	UpdatedAt           int64
	Size                int64
	ThumbSize           int64
	User                User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupID             *uint64 // can be null
	Group               Group   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	BucketID            uint64
	Bucket              storage.Bucket `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	PlaceID             *uint64
	Place               Place    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name                string   `gorm:"type:varchar(300)"`
	MimeType            string   `gorm:"type:varchar(50)"`
	GpsLat              *float64 `gorm:"type:double"`
	GpsLong             *float64 `gorm:"type:double"`
	Favourite           bool
	Deleted             bool `gorm:"index:user_asset_created,priority:2;not null;default 0"`
	Width               uint16
	Height              uint16
	ThumbWidth          uint16
	ThumbHeight         uint16
	Duration            uint32
	Processed           bool `gorm:"not null;default 0"`
	PresignedUntil      int64
	PresignedURL        string `gorm:"type:varchar(2000)"`
	PresignedThumbUntil int64
	PresignedThumbURL   string `gorm:"type:varchar(2000)"`
}

// GetPath returns the path of the asset. For example:
//   - group/56/image.jpg
//   - user/3/file.xls
func (a *Asset) GetPath() string {
	return a.GetPathOrThumb(false)
}

func (a *Asset) GetThumbPath() string {
	return a.GetPathOrThumb(true)
}

func (a *Asset) GetPathOrThumb(thumb bool) string {
	subDir := ""
	if a.GroupID != nil {
		// This is an asset uploaded to a Group (as part of a Post)
		subDir = "group/" + strconv.FormatUint(*a.GroupID, 10)
	} else {
		// It must be an asset for a User (private or part of Post on their "wall")
		subDir = "user/" + strconv.FormatUint(a.UserID, 10)
	}
	path := subDir + "/" + strconv.FormatUint(a.ID, 10)
	if thumb {
		// Thumbs are always JPEG
		path += "_thumb.jpg"
	} else {
		path += strings.ToLower(filepath.Ext(a.Name))
	}
	return path
}

func (a *Asset) BeforeSave(tx *gorm.DB) (err error) {
	// Restrict the characters in Name
	var name strings.Builder
	for i, c := range a.Name {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			(c == '.' && i > 0) || (c == '-') || (c == '_') {

			name.WriteRune(c)
		} else {
			// Replace all other characters with '_' (underscore)
			name.WriteString("_")
		}
	}
	a.Name = name.String()
	return
}

func (a *Asset) GetRoughLocation() (location Location) {
	if a.GpsLat != nil && a.GpsLong != nil {
		// Truncate - only use 0.0001 of precision
		location.GpsLat = float64(int(*a.GpsLat*10000)) / 10000
		location.GpsLong = float64(int(*a.GpsLong*10000)) / 10000
	}
	return
}

// CreateUploadURI creates a URI that is then to be called by the App
// The URI could be either:
//  1. local (i.e. starting with /..)
//  2. Pre-signed remote S3 upload URI
//
// TODO: Add error response
func (a *Asset) CreateUploadURI(thumb bool) string {
	// TODO: Better way?
	if a.Bucket.ID != a.BucketID {
		db.Instance.Preload("Bucket").First(a)
	}
	if a.Bucket.IsS3() {
		return a.Bucket.CreateS3UploadURI(a.GetPathOrThumb(thumb))
	}
	return "/backup/upload?id=" + strconv.FormatUint(a.ID, 10) + "thumb=" + strconv.FormatBool(thumb)
}

// NOTE: a.Bucket must be preloaded
func (a *Asset) GetS3DownloadURL(thumb bool) (string, int64) {
	// Separatel fields for thumb...
	if thumb {
		if a.PresignedThumbURL == "" || a.PresignedThumbUntil < time.Now().Add(presignValidAtLeastFor).Unix() {
			// Need to sign again..
			a.PresignedThumbURL = a.Bucket.CreateS3DownloadURI(a.GetPathOrThumb(thumb), presignViewURLFor)
			a.PresignedThumbUntil = time.Now().Add(presignViewURLFor).Unix()
			db.Instance.Updates(a)
		}
		return a.PresignedThumbURL, a.PresignedThumbUntil
	}

	// Valid at least for another 30 minutes?
	if a.PresignedURL == "" || a.PresignedUntil < time.Now().Add(presignValidAtLeastFor).Unix() {
		// Need to sign again..
		a.PresignedURL = a.Bucket.CreateS3DownloadURI(a.GetPathOrThumb(thumb), presignViewURLFor)
		a.PresignedUntil = time.Now().Add(presignViewURLFor).Unix()
		db.Instance.Updates(a)
	}
	return a.PresignedURL, a.PresignedUntil
}

// func (a *Asset) AfterSave(tx *gorm.DB) (err error) {
// 	// Scan the thumb for faces
// 	if a.ThumbSize <= 0 {
// 		return
// 	}
// 	foundFaces, err := faces.ProcessPhoto("/mnt/data1/circled-data/" + a.GetThumbPath())
// 	if err != nil {
// 		log.Print(err)
// 		return
// 	}
// 	//fmt.Printf("Asset: %d, num faces: %d; saving...\n", a.ID, len(foundFaces))
// 	for _, face := range foundFaces {
// 		desc := [128]float32(face.Descriptor)
// 		faceObject := Face{
// 			AssetID:    a.ID,
// 			Descriptor: utils.Float32ArrayToByteArray(desc[:]),
// 			RectX1:     uint16(face.Rectangle.Min.X),
// 			RectY1:     uint16(face.Rectangle.Min.Y),
// 			RectX2:     uint16(face.Rectangle.Max.X),
// 			RectY2:     uint16(face.Rectangle.Max.Y),
// 		}
// 		db.Instance.Save(&faceObject)
// 	}
// 	return
// }
