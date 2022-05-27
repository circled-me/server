package storage

import (
	"fmt"
	"io"
	"server/db"
)

type StorageAPI interface {
	Save(path string, reader io.Reader) (int64, error)
	Delete(path string) error
	GetTotalSpace() uint64
	GetFreeSpace() uint64
	GetBucket() *Bucket
	// CreateSubDir(dir string) error
}

type Storage struct {
	StorageAPI
	TotalSpace uint64
	FreeSpace  uint64
	Bucket     Bucket
}

var (
	cachedStorage []StorageAPI
)

func Init() {
	db.Instance.AutoMigrate(&Bucket{})

	cachedStorage = []StorageAPI{}
	var buckets []Bucket
	err := db.Instance.Find(&buckets).Error
	if err != nil {
		panic(err)
	}
	fmt.Printf("Storage Buckets found: %d\n", len(buckets))
	var storage StorageAPI
	for _, bucket := range buckets {
		fmt.Printf("Bucket: %+v\n", bucket)
		if bucket.StorageType == StorageTypeFile {
			storage = NewDiskStorage(&bucket)
		} else if bucket.StorageType == StorageTypeS3 {
			storage = NewS3Storage(&bucket)
		} else {
			panic(fmt.Sprintf("Storage type unavailable for Bucket %d", bucket.ID))
		}
		cachedStorage = append(cachedStorage, storage)
	}
}

func (s *Storage) GetTotalSpace() uint64 {
	return s.TotalSpace
}

func (s *Storage) GetFreeSpace() uint64 {
	return s.FreeSpace
}

func (s *Storage) GetBucket() *Bucket {
	return &s.Bucket
}

func StorageFrom(bucket *Bucket) StorageAPI {
	for _, s := range cachedStorage {
		if s.GetBucket().ID == bucket.ID {
			return s
		}
	}
	return nil
}

func GetDefaultStorage() StorageAPI {
	if len(cachedStorage) == 0 {
		panic("no storage available")
	}
	for _, s := range cachedStorage {
		if s.GetBucket().StorageType == StorageTypeFile {
			return s
		}
	}
	for _, s := range cachedStorage {
		return s
	}
	return nil // Cannot reach here
}
