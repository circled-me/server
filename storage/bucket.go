package storage

import (
	"log"
	"net/url"
	"os"
	"server/db"
	"strings"
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
	Path        string // Path on a drive or a URL with prefix (for S3 buckets)
	AuthDetails string // Authentication details. In case of S3 bucket - "region:key:secret"
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

// TODO: Do not create session, etc twice (for main and thumb separately)
func (b *Bucket) CreateS3UploadURI(path string) string {
	auth := strings.SplitN(b.AuthDetails, ":", 3)
	if len(auth) != 3 {
		log.Printf("Invalid auth details for bucket: %d", b.ID)
		return "Invalid auth details"
	}
	if auth[0] == "" {
		auth[0] = "us-east-1"
	}
	u, _ := url.Parse(b.Path)
	//prefix := strings.Trim(u.Path, "/")
	sess, _ := session.NewSession(&aws.Config{
		Region: &auth[0],
		//S3ForcePathStyle: aws.Bool(false), // TODO: Should be flexible
		Endpoint:    aws.String(u.String()),
		Credentials: credentials.NewStaticCredentials(auth[1], auth[2], ""),
	})
	u.Path = ""
	svc := s3.New(sess)
	req, _ := svc.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String("nik-test-1"),
		Key:    aws.String(path),
	})
	out, err := req.Presign(15 * time.Minute)
	if err != nil {
		log.Printf("Cannot sign request: %v", err)
		return err.Error()
	}
	return out
}

func (b *Bucket) CreateS3DownloadURI(path string, expiry time.Duration) string {
	// TODO: Improve this below and above
	auth := strings.SplitN(b.AuthDetails, ":", 3)
	if len(auth) != 3 {
		log.Printf("Invalid auth details for bucket: %d", b.ID)
		return "Invalid auth details"
	}
	if auth[0] == "" {
		auth[0] = "us-east-1"
	}
	u, _ := url.Parse(b.Path)
	//prefix := strings.Trim(u.Path, "/")
	sess, _ := session.NewSession(&aws.Config{
		Region:      &auth[0],
		Endpoint:    aws.String(u.String()),
		Credentials: credentials.NewStaticCredentials(auth[1], auth[2], ""),
	})
	u.Path = ""
	log.Print("Get URL: " + u.String())
	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String("nik-test-1"),
		Key:    aws.String(path),
	})
	out, err := req.Presign(expiry)
	if err != nil {
		log.Printf("Cannot sign request 2: %v", err)
		return err.Error()
	}
	return out
}
