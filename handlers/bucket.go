package handlers

import (
	"log"
	"net/http"
	"server/db"
	"server/models"
	"server/storage"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func hasWriteAccess(bucket *storage.Bucket) error {
	storage := storage.NewStorage(bucket)
	testPath := "tmp/path"
	_, err := storage.Save(testPath, strings.NewReader("some-content"))
	if err != nil {
		log.Printf("Cannot save to bucket: %+v", bucket)
		return err
	}
	err = storage.UpdateRemoteFile(testPath, "text/plain")
	if err != nil {
		log.Printf("Cannot update bucket: %+v", bucket)
		return err
	}
	err = storage.Delete(testPath)
	if err != nil {
		log.Printf("Cannot delete: %+v", bucket)
		return err
	}
	err = storage.DeleteRemoteFile(testPath)
	if err != nil {
		log.Printf("Cannot delete remote object from bucket: %+v", bucket)
		return err
	}
	return nil
}

func cleanupPath(in *storage.Bucket) {
	for strings.Contains(in.Path, "..") {
		in.Path = strings.ReplaceAll(in.Path, "..", "")
	}
	for strings.Contains(in.Path, "//") {
		in.Path = strings.ReplaceAll(in.Path, "//", "/")
	}
}

func BucketSave(c *gin.Context, user *models.User) {
	bucket := storage.Bucket{}
	err := c.ShouldBindWith(&bucket, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	cleanupPath(&bucket)

	if bucket.Name == "" {
		c.JSON(http.StatusBadRequest, Response{"Empty bucket name"})
		return
	}
	if bucket.StorageType == storage.StorageTypeFile {
		if bucket.Path == "" {
			c.JSON(http.StatusBadRequest, Response{"Empty bucket path"})
			return
		}
		if bucket.Path[0] != '/' {
			c.JSON(http.StatusBadRequest, Response{"Path must be absolute and start with / (slash)"})
			return
		}
	} else if bucket.StorageType == storage.StorageTypeS3 {
		if bucket.S3Key == "" || bucket.S3Secret == "" {
			c.JSON(http.StatusBadRequest, Response{"'S3 Key' and 'S3 Secret' must be provided"})
			return
		}
		if bucket.Region == "" {
			bucket.Region = "us-east-1"
		}
	} else {
		c.JSON(http.StatusBadRequest, Response{"'type' must be one of 'file' or 's3'"})
		return
	}
	if err := hasWriteAccess(&bucket); err != nil {
		c.JSON(http.StatusForbidden, Response{"No write access to bucket: " + err.Error()})
		return
	}
	if err = bucket.TryInit(); err != nil {
		c.JSON(http.StatusForbidden, Response{err.Error()})
		return
	}
	if bucket.ID == 0 {
		err = db.Instance.Create(&bucket).Error
	} else {
		err = db.Instance.Save(&bucket).Error
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
	// Re-initialize storage
	storage.Init()
	c.JSON(http.StatusOK, OKResponse)
}

func BucketList(c *gin.Context, user *models.User) {
	buckets := []storage.Bucket{}
	result := db.Instance.Find(&buckets)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, buckets)
}
