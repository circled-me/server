package storage

import (
	"io"
	"net/http"
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

func (s *S3Storage) Load(path string, writer io.Writer) (int64, error) {
	// TODO
	return 0, nil
}

func (s *S3Storage) Serve(path string, request *http.Request, writer http.ResponseWriter) {
	// TODO
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
