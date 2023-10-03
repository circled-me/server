package storage

import (
	"errors"
	"log"
	"net/url"
	"server/db"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/sys/unix"
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
	ID               uint64      `gorm:"primaryKey" json:"id"`
	CreatedAt        int         `json:"-"`
	UpdatedAt        int         `json:"-"`
	Name             string      `gorm:"type:varchar(200)" json:"name"`
	AssetPathPattern string      `gorm:"type:varchar(200)" json:"asset_path_pattern"`
	StorageType      StorageType `json:"storage_type"`
	Path             string      `gorm:"type:varchar(300)" json:"path"`     // Path on a drive or a prefix (for S3 buckets)
	Endpoint         string      `gorm:"type:varchar(300)" json:"endpoint"` // URL for S3 buckets; if empty - defaults to AWS S3
	S3Key            string      `gorm:"type:varchar(200)" json:"s3key"`
	S3Secret         string      `gorm:"type:varchar(200)" json:"s3secret"`
	Region           string      `gorm:"type:varchar(20)" json:"s3region"`     // Defaults to us-east-1
	SSEEncryption    string      `gorm:"type:varchar(20)" json:"s3encryption"` // Server-side encryption (or empty for no encryption)
}

func (b *Bucket) IsS3() bool {
	return b.StorageType == StorageTypeS3
}

func (b *Bucket) CanSave() (err error) {
	if b.ID > 0 {
		count := int64(0)
		if db.Instance.Raw("select exists(select id from assets where deleted=0 and bucket_id=?)", b.ID).Scan(&count).Error != nil {
			return errors.New("DB error")
		}
		if count != 0 {
			return errors.New("Cannot modify bucket as it is already in use")
		}
	}
	if b.StorageType == StorageTypeS3 {
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
	input := &s3.PutObjectInput{
		Bucket: &b.Name,
		Key:    aws.String(b.GetRemotePath(path)),
	}
	if b.SSEEncryption != "" {
		input.ServerSideEncryption = &b.SSEEncryption
	}
	req, _ := svc.PutObjectRequest(input)
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
		Key:    aws.String(b.GetRemotePath(path)),
	})
	out, err := req.Presign(expiry)
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

func (b *Bucket) GetUsage() int64 {
	result := int64(-1)
	if err := db.Instance.Raw("select ifnull(sum(size+thumb_size), 0) from assets where bucket_id=? and deleted=0", b.ID).Scan(&result).Error; err != nil {
		return -1
	}
	return result
}

func (b *Bucket) GetSpaceInfo() (available, size int64) {
	if b.StorageType != StorageTypeFile {
		return -1, -1 // unknown for S3 or any other storage type
	}
	var stat unix.Statfs_t
	unix.Statfs(b.Path, &stat)
	return int64(stat.Bavail) * int64(stat.Bsize), int64(stat.Blocks) * int64(stat.Bsize)
}
