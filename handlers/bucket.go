package handlers

import (
	"log"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/storage"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type BucketCreateRequest struct {
	Name          string `form:"name" binding:"required"`
	Type          string `form:"type" binding:"required"` // 'file' or 's3'
	Path          string `form:"path"`
	Endpoint      string `form:"endpoint"`
	S3Key         string `form:"s3key"`
	S3Secret      string `form:"s3secret"`
	Region        string `form:"region"`
	SSEEncryption string `form:"sseEncryption"`
}

func hasWriteAccess(bucket *storage.Bucket) error {
	storage := storage.NewStorage(bucket)
	testPath := "tmp/path"
	_, err := storage.Save(testPath, strings.NewReader("some-content"))
	if err != nil {
		log.Printf("Cannot save to bucket: %+v", bucket)
		return err
	}
	err = storage.UpdateFile(testPath, "text/plain")
	if err != nil {
		log.Printf("Cannot update bucket: %+v", bucket)
		return err
	}
	err = storage.Delete(testPath)
	if err != nil {
		log.Printf("Cannot delete from bucket: %+v", bucket)
		return err
	}
	return nil
}

func cleanupPath(in *BucketCreateRequest) {
	for strings.Contains(in.Path, "..") {
		in.Path = strings.ReplaceAll(in.Path, "..", "")
	}
	for strings.Contains(in.Path, "//") {
		in.Path = strings.ReplaceAll(in.Path, "//", "/")
	}
}

func BucketCreate(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	postReq := BucketCreateRequest{}
	err := c.ShouldBindWith(&postReq, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cleanupPath(&postReq)

	bucket := storage.Bucket{
		Name: postReq.Name,
		Path: postReq.Path,
	}
	if postReq.Type == "file" {
		bucket.StorageType = storage.StorageTypeFile
		if postReq.Path == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Empty bucket path"})
			return
		}
		if postReq.Path[0] != '/' {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Path must be absolute and start with / (slash)"})
			return
		}
	} else if postReq.Type == "s3" && postReq.S3Key != "" && postReq.S3Secret != "" {
		bucket.StorageType = storage.StorageTypeS3
		bucket.S3Key = postReq.S3Key
		bucket.S3Secret = postReq.S3Secret
		bucket.Endpoint = postReq.Endpoint
		bucket.Region = postReq.Region
		bucket.SSEEncryption = postReq.SSEEncryption
		if bucket.Region == "" {
			bucket.Region = "us-east-1"
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'type' must be one of 'file' or 's3'; if 'type' is 's3', then 's3key', 's3secret' must be provided too ('region' and 'endpoint' also configurable)"})
		return
	}
	if err := hasWriteAccess(&bucket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No access to bucket: " + err.Error()})
		return
	}
	if err = bucket.Create(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Re-initialize storage
	storage.Init()
	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func BucketList(c *gin.Context) {
	// session := auth.LoadSession(c)
	// user := session.User()
	// if user.ID == 0 || !user.HasPermission(models.PermissionAdmin) {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
	// 	return
	// }
	buckets := []storage.Bucket{}
	result := db.Instance.Find(&buckets)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	c.JSON(http.StatusOK, buckets)
}
