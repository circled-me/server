package models

import (
	"path/filepath"
	"server/storage"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

type Asset struct {
	ID        uint64 `gorm:"primaryKey"`
	UserID    uint64 `gorm:"index:uniq_remote_id,unique;not null"`
	RemoteID  string `gorm:"type:varchar(300);index:uniq_remote_id,unique;not null"`
	CreatedAt int
	UpdatedAt int
	Size      int64
	User      User    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	GroupID   *uint64 // can be null
	Group     Group   `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	BucketID  uint64
	Bucket    storage.Bucket `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Name      string         `gorm:"type:varchar(300)"`
	MimeType  string         `gorm:"type:varchar(50)"`
}

// GetPath returns the path of the asset. For example:
//  - group/56/image.jpg
//  - user/3/file.xls
func (a *Asset) GetPath() string {
	subDir := ""
	if a.GroupID != nil {
		// This is an asset uploaded to a Group (as part of a Post)
		subDir = "group/" + strconv.FormatUint(*a.GroupID, 10)
	} else {
		// It must be an asset for a User (private or part of Post on their "wall")
		subDir = "user/" + strconv.FormatUint(a.UserID, 10)
	}
	return subDir + "/" + strconv.FormatUint(a.ID, 10) + filepath.Ext(a.Name)
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
	return
}
