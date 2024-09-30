package models

import (
	"fmt"
	"path/filepath"
	"server/config"
	"server/db"
	"server/storage"
	"strconv"
	"strings"
	"time"

	"github.com/zsefvlol/timezonemapper"
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
	TimeOffset          *int     `gorm:"type:int"`
	Favourite           bool
	Deleted             bool `gorm:"index:user_asset_created,priority:2;not null;default 0"`
	Width               uint16
	Height              uint16
	ThumbWidth          uint16
	ThumbHeight         uint16
	Duration            uint32
	Path                string `gorm:"type:varchar(2048)"` // Full path of the asset, including file/object name
	ThumbPath           string `gorm:"type:varchar(2048)"` // Same but for thumbnail
	PresignedUntil      int64
	PresignedURL        string `gorm:"type:varchar(2000)"`
	PresignedThumbUntil int64
	PresignedThumbURL   string `gorm:"type:varchar(2000)"`
}

// CreatePath returns new path for an asset. For example:
//   - group/56/image.jpg
//   - user/3/file.xls
func (a *Asset) CreatePath() string {
	return a.CreatePathOrThumb(false)
}

func (a *Asset) CreateThumbPath() string {
	return a.CreatePathOrThumb(true)
}

func (a *Asset) GetCreatedTimeInLocation() time.Time {
	if a.GpsLat == nil || a.GpsLong == nil {
		return time.Unix(a.CreatedAt, 0)
	}
	zone, err := time.LoadLocation(timezonemapper.LatLngToTimezoneString(*a.GpsLat, *a.GpsLong))
	if err != nil {
		return time.Unix(a.CreatedAt, 0)
	}
	return time.Unix(a.CreatedAt, 0).In(zone)
}

func (a *Asset) getAssetFilePathNoExt() string {
	result := a.Bucket.AssetPathPattern
	if result == "" {
		result = config.DEFAULT_ASSET_PATH_PATTERN
	}
	assetTime := a.GetCreatedTimeInLocation()
	ext := filepath.Ext(a.Name)
	name := a.Name[:len(a.Name)-len(ext)] // remove extension
	result = strings.ReplaceAll(result, "<id>", strconv.FormatUint(a.ID, 10))
	result = strings.ReplaceAll(result, "<name>", name)
	result = strings.ReplaceAll(result, "<year>", strconv.Itoa(assetTime.Year()))
	result = strings.ReplaceAll(result, "<month>", fmt.Sprintf("%02d", assetTime.Month()))
	result = strings.ReplaceAll(result, "<Month>", assetTime.Month().String())
	return result
}

func (a *Asset) CreatePathOrThumb(thumb bool) string {
	subDir := ""
	if a.GroupID != nil {
		// This is an asset uploaded to a Group (as part of a Post)
		subDir = "group/" + strconv.FormatUint(*a.GroupID, 10)
	} else {
		// It must be an asset for a User (private or part of Post on their "wall")
		subDir = "user/" + strconv.FormatUint(a.UserID, 10)
	}
	path := subDir + "/" + a.getAssetFilePathNoExt()
	// Add extension
	if thumb {
		// Thumbs are always JPEG
		path += "_thumb.jpg"
	} else {
		path += strings.ToLower(filepath.Ext(a.Name))
	}
	return path
}

func (a *Asset) GetPathOrThumb(thumb bool) string {
	if thumb {
		if a.ThumbPath == "" {
			a.ThumbPath = a.CreateThumbPath()
		}
		return a.ThumbPath
	}
	if a.Path == "" {
		a.Path = a.CreatePath()
	}
	return a.Path
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

func (a *Asset) IsVideo() bool {
	return strings.HasPrefix(strings.ToLower(a.MimeType), "video/")
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
//  2. Pre-signed remote S3 upload URL
//
// TODO: Add error response
func (a *Asset) CreateUploadURI(thumb bool, webToken string) string {
	// TODO: Better way?
	if a.Bucket.ID != a.BucketID {
		db.Instance.Preload("Bucket").First(a)
	}
	if a.Bucket.IsS3() {
		return a.Bucket.CreateS3UploadURI(a.GetPathOrThumb(thumb))
	}
	if webToken != "" {
		return "/w/upload/" + webToken + "/?id=" + strconv.FormatUint(a.ID, 10) + "&thumb=" + strconv.FormatBool(thumb)
	}
	return "/backup/upload?id=" + strconv.FormatUint(a.ID, 10) + "&thumb=" + strconv.FormatBool(thumb)
}

// NOTE: a.Bucket must be preloaded
func (a *Asset) GetS3DownloadURL(thumb bool) (string, int64) {
	// Separatel fields for thumb...
	if thumb && a.ThumbSize > 0 {
		if a.PresignedThumbURL == "" || a.PresignedThumbUntil < time.Now().Add(presignValidAtLeastFor).Unix() {
			// Need to sign again..
			a.PresignedThumbURL = a.Bucket.CreateS3DownloadURI(a.ThumbPath, presignViewURLFor)
			a.PresignedThumbUntil = time.Now().Add(presignViewURLFor).Unix()
			db.Instance.Updates(a)
		}
		return a.PresignedThumbURL, a.PresignedThumbUntil
	}

	// Valid at least for another 30 minutes?
	if a.PresignedURL == "" || a.PresignedUntil < time.Now().Add(presignValidAtLeastFor).Unix() {
		// Need to sign again..
		a.PresignedURL = a.Bucket.CreateS3DownloadURI(a.Path, presignViewURLFor)
		a.PresignedUntil = time.Now().Add(presignViewURLFor).Unix()
		db.Instance.Updates(a)
	}
	return a.PresignedURL, a.PresignedUntil
}
