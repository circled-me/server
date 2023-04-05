package storage

import (
	"log"
	"net/url"
	"os"
	"server/db"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
	Path        string `gorm:"type:varchar(300)"` // Path on a drive or a prefix (for S3 buckets)
	Endpoint    string `gorm:"type:varchar(300)"` // URL for S3 buckets; if empty - defaults to AWS S3
	S3Key       string `gorm:"type:varchar(200)"`
	S3Secret    string `gorm:"type:varchar(200)"`
	Region      string `gorm:"type:varchar(20)"` // Defaults to us-east-1
}

func (b *Bucket) IsS3() bool {
	return b.StorageType == StorageTypeS3
}

func (b *Bucket) Create() (err error) {
	err = db.Instance.Create(b).Error
	if err != nil {
		return
	}
	if b.StorageType == StorageTypeFile {
		// Pre-create locations on disk
		if err = os.MkdirAll(b.Path+StorageLocationUser, 0777); err != nil {
			return
		}
		if err = os.MkdirAll(b.Path+StorageLocationGroup, 0777); err != nil {
			return
		}
	} else if b.StorageType == StorageTypeS3 {
		_, err = url.Parse(b.Path)
	}
	return
}

func (b *Bucket) GetRemotePath(path string) string {
	return b.Path + "/" + path
}

// TODO: Do not create session, etc twice (for main and thumb separately)
func (b *Bucket) CreateS3UploadURI(path string) string {
	svc := b.CreateSVC()
	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: &b.Name,
		Key:    aws.String(b.GetRemotePath(path)),
	})
	out, err := req.Presign(15 * time.Minute)
	if err != nil {
		log.Printf("Cannot sign request: %v", err)
		return err.Error()
	}
	return out
}

func (b *Bucket) CreateS3DownloadURI(path string, expiry time.Duration) string {
	svc := b.CreateSVC()
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: &b.Name,
		Key:    aws.String(b.Path + "/" + path),
	})
	out, err := req.Presign(expiry)
	log.Printf("Download URI: %v, %v\n", err, out)
	if err != nil {
		log.Printf("Cannot sign request 2: %v", err)
		return err.Error()
	}
	return out
}

func (b *Bucket) CreateSVC() *s3.S3 {
	config := &aws.Config{
		Region:      &b.Region,
		Credentials: credentials.NewStaticCredentials(b.S3Key, b.S3Secret, ""),
	}
	if b.Endpoint != "" {
		config.Endpoint = &b.Endpoint
	}
	sess, _ := session.NewSession(config)
	return s3.New(sess)
}
