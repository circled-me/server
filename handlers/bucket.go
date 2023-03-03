package handlers

import (
	"net/http"
	"server/storage"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type BucketCreateRequest struct {
	Name string `form:"name" binding:"required"`
	Path string `form:"path" binding:"required"`
	Type string `form:"type" binding:"required"` // 'file' or 's3'
	Auth string `form:"auth"`
}

func BucketCreate(c *gin.Context) {
	// TODO: this should be TOKEN protected
	// session := auth.LoadSession(c)
	// user := session.User()
	// if user.ID == 0 || !user.HasPermission(models.PermissionAdmin) {
	// 	c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
	// 	return
	// }
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
	} else if postReq.Type == "s3" && postReq.Auth != "" {
		bucket.StorageType = storage.StorageTypeS3
		bucket.AuthDetails = postReq.Auth // TODO: validation + test request
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "'type' must be one of 'file' or 's3'; if 'type' is 's3', then 'auth' details must be provided too ('region:key:secret')"})
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
