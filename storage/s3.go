package storage

import (
	"io"
)

type S3Storage struct {
	Storage
}

// func (s *S3Storage) CreateSubDir(dir string) error {
// 	// N/A
// 	return nil
// }

func (s *S3Storage) Save(path string, reader io.Reader) (int64, error) {
	// TODO
	return 0, nil
}

func (s *S3Storage) Delete(path string) error {
	// TODO
	return nil
}

func NewS3Storage(bucket *Bucket) StorageAPI {
	return &S3Storage{
		Storage: Storage{
			Bucket: *bucket,
		},
	}
}
