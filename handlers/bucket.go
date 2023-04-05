package handlers

import (
	"net/http"
	"server/auth"
	"server/models"
	"server/storage"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type BucketCreateRequest struct {
	Name          string `form:"name" binding:"required"`
	Type          string `form:"type" binding:"required"` // 'file' or 's3'
	Path          string `form:"path" binding:"required"`
	Endpoint      string `form:"endpoint"`
	S3Key         string `form:"s3key"`
	S3Secret      string `form:"s3secret"`
	Region        string `form:"region"`
	SSEEncryption string `form:"sseEncryption"`
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
	bucket := storage.Bucket{
		Name: postReq.Name,
		Path: postReq.Path,
	}
	if postReq.Type == "file" {
		bucket.StorageType = storage.StorageTypeFile
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
		// TODO: validation + test request
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'type' must be one of 'file' or 's3'; if 'type' is 's3', then 's3key', 's3secret' must be provided too ('region' and 'endpoint' also configurable)"})
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
