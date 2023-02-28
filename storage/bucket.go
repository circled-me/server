package storage

import (
	"os"
	"server/db"
)

type StorageType uint8

const (
	StorageTypeFile StorageType = 0
	StorageTypeS3   StorageType = 1
)
const (
	StorageLocationUser  = "/user"
	StorageLocationGroup = "/group"
)

type Bucket struct {
	ID          uint64 `gorm:"primaryKey"`
	CreatedAt   int
	UpdatedAt   int
	Name        string `gorm:"type:varchar(200)"`
	StorageType StorageType
	Path        string // Path on a drive or a prefix in a S3 bucket
	AuthDetails string // Authentication details. In case of S3 bucket - "key:secret"
}

func (b *Bucket) Create() error {
	err := db.Instance.Create(b).Error
	if err != nil {
		return err
	}
	if b.StorageType == StorageTypeFile {
		// Pre-create locations on disk
		if err = os.MkdirAll(b.Path+StorageLocationUser, 0777); err != nil {
			return err
		}
		if err = os.MkdirAll(b.Path+StorageLocationGroup, 0777); err != nil {
			return err
		}
	}
	return nil
}
