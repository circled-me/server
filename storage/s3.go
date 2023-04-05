package storage

import (
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3Storage struct {
	Storage
	s3Client *s3.S3
}

// GetFullPath returns local temp path in case of S3
func (s *S3Storage) GetFullPath(path string) string {
	// TODO: the temp directory should be configurable
	return "/tmp/" + strings.ReplaceAll(path, "/", "_")
}

func (s *S3Storage) EnsureDirExists(dir string) error {
	return nil
}

func NewS3Storage(bucket *Bucket) StorageAPI {
	result := &S3Storage{
		Storage: Storage{
			Bucket: *bucket,
		},
		s3Client: bucket.CreateSVC(),
	}
	result.specifics = result
	return result
}

// EnsureLocalFile downloads a S3 object locally
func (s *S3Storage) EnsureLocalFile(path string) error {
	// S3 request
	resp, err := s.s3Client.GetObject(&s3.GetObjectInput{
		Bucket: &s.Bucket.Name,
		Key:    aws.String(s.Bucket.GetRemotePath(path)),
	})
	if err != nil {
		return err
	}
	// Lcoal file
	out, err := os.Create(s.GetFullPath(path))
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func (s *S3Storage) ReleaseLocalFile(path string) {
	s.Delete(path)
}

// UpdateFile updates the remote S3 object (uploads the local copy)
func (s *S3Storage) UpdateFile(path, mimeType string) error {
	data, err := os.Open(s.GetFullPath(path))
	if err != nil {
		return err
	}
	defer data.Close()

	uploader := s3manager.NewUploaderWithClient(s.s3Client)
	input := s3manager.UploadInput{
		Bucket:      &s.Bucket.Name,
		Key:         aws.String(s.Bucket.GetRemotePath(path)),
		ContentType: &mimeType,
		Body:        data,
	}
	// TODO: Option for SSE
	// input.ServerSideEncryption = &config.S3Encryption
	_, err = uploader.Upload(&input)

	return err
}

func (s *S3Storage) DeleteRemoteFile(path string) {
	_, err := s.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: &s.Bucket.Name,
		Key:    aws.String(s.Bucket.GetRemotePath(path)),
	})
	if err != nil {
		log.Printf("Error deleting remote object %s: %s", s.Bucket.GetRemotePath(path), err.Error())
	}
}
