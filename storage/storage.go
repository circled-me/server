package storage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"server/db"
)

type StorageSpecificAPI interface {
	GetFullPath(path string) string
	EnsureDirExists(dir string) error
	EnsureLocalFile(path string) error
	ReleaseLocalFile(path string)
	DeleteRemoteFile(path string)
	UpdateFile(path, mimeType string) error
}

type StorageAPI interface {
	StorageSpecificAPI

	GetSize(path string) int64
	Save(path string, reader io.Reader) (int64, error)
	Load(path string, writer io.Writer) (int64, error)
	Serve(path string, request *http.Request, writer http.ResponseWriter)
	Delete(path string) error
	GetTotalSpace() uint64
	GetFreeSpace() uint64
	GetBucket() *Bucket
}

type Storage struct {
	StorageAPI
	specifics  StorageAPI
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
	log.Printf("Storage Buckets found: %d\n", len(buckets))
	var storage StorageAPI
	for _, bucket := range buckets {
		log.Printf("Bucket: %+v\n", bucket)
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

//
// NOTE: All the functions below work on a local file
//

func (s *Storage) GetSize(path string) int64 {
	fi, err := os.Stat(s.GetFullPath(path))
	if err != nil {
		return -1
	}
	return fi.Size()
}

func (s *Storage) Save(path string, reader io.Reader) (int64, error) {
	fileName := s.GetFullPath(path)
	if err := s.EnsureDirExists(filepath.Dir(fileName)); err != nil {
		return 0, err
	}
	file, err := os.Create(fileName)
	if err != nil {
		return 0, err
	}
	result, err := io.Copy(file, reader)
	file.Close()
	return result, err
}

func (s *Storage) Load(path string, writer io.Writer) (int64, error) {
	fileName := s.GetFullPath(path)
	file, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	result, err := io.Copy(writer, file)
	file.Close()
	return result, err
}

func (s *Storage) Serve(path string, request *http.Request, writer http.ResponseWriter) {
	fileName := s.GetFullPath(path)
	http.ServeFile(writer, request, fileName)
}

func (s *Storage) Delete(path string) error {
	return os.Remove(s.GetFullPath(path))
}

//
// Proxy methods
//

func (s *Storage) GetFullPath(path string) string {
	return s.specifics.GetFullPath(path)
}
func (s *Storage) EnsureDirExists(dir string) error {
	return s.specifics.EnsureDirExists(dir)
}
func (s *Storage) EnsureLocalFile(path string) error {
	return s.specifics.EnsureLocalFile(path)
}
func (s *Storage) ReleaseLocalFile(path string) {
	s.specifics.ReleaseLocalFile(path)
}
func (s *Storage) DeleteRemoteFile(path string) {
	s.specifics.DeleteRemoteFile(path)
}
func (s *Storage) UpdateFile(path, mimeType string) error {
	return s.specifics.UpdateFile(path, mimeType)
}
