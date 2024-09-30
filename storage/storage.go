package storage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"server/config"
	"server/db"
)

type StorageSpecificAPI interface {
	// GetFullPath always returns local path (tmp path for remote storage)
	GetFullPath(path string) string
	EnsureDirExists(dir string) error
	EnsureLocalFile(path string) error
	ReleaseLocalFile(path string)
	DeleteRemoteFile(path string) error
	UpdateRemoteFile(path, mimeType string) error
}

type StorageAPI interface {
	StorageSpecificAPI

	GetSize(path string) int64
	Save(path string, reader io.Reader) (int64, error)
	Load(path string, writer io.Writer) (int64, error)
	Serve(path string, request *http.Request, writer http.ResponseWriter)
	Delete(path string) error
	GetBucket() *Bucket
}

type Storage struct {
	StorageAPI
	specifics StorageAPI
	Bucket    Bucket
}

var (
	cachedStorage []StorageAPI
)

func Init() {
	if err := db.Instance.AutoMigrate(&Bucket{}); err != nil {
		log.Printf("Auto-migrate error: %v", err)
	}

	cachedStorage = []StorageAPI{}
	var buckets []Bucket
	err := db.Instance.Find(&buckets).Error
	if err != nil {
		panic(err)
	}
	if len(buckets) == 0 {
		log.Printf("No Storage Buckets found")
		// Create default bucket if DEFAULT_BUCKET_DIR is set
		if config.DEFAULT_BUCKET_DIR != "" {
			log.Printf("Creating default bucket in directory: %s", config.DEFAULT_BUCKET_DIR)
			bucket := Bucket{
				Name:             "Main",
				Path:             config.DEFAULT_BUCKET_DIR,
				AssetPathPattern: config.DEFAULT_ASSET_PATH_PATTERN,
				StorageType:      StorageTypeFile,
			}
			err := db.Instance.Create(&bucket).Error
			if err != nil {
				log.Fatalf("Error creating default bucket: %v", err)
			}
			log.Printf("Default bucket created!")
			// Reload buckets
			_ = db.Instance.Find(&buckets)
		}
	}
	for _, bucket := range buckets {
		log.Printf("Bucket: %+v\n", bucket)
		storage := NewStorage(&bucket)
		cachedStorage = append(cachedStorage, storage)
	}
}

func NewStorage(bucket *Bucket) StorageAPI {
	if bucket.StorageType == StorageTypeFile {
		return NewDiskStorage(bucket)
	} else if bucket.StorageType == StorageTypeS3 {
		return NewS3Storage(bucket)
	} else {
		panic(fmt.Sprintf("Storage type unavailable for Bucket %d", bucket.ID))
	}
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
	for _, s := range cachedStorage {
		if s.GetBucket().StorageType == StorageTypeFile {
			return s
		}
	}
	for _, s := range cachedStorage {
		return s
	}
	return nil
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
func (s *Storage) DeleteRemoteFile(path string) error {
	return s.specifics.DeleteRemoteFile(path)
}
func (s *Storage) UpdateRemoteFile(path, mimeType string) error {
	return s.specifics.UpdateRemoteFile(path, mimeType)
}
