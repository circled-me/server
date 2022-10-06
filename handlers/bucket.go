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
	Name string `form:"name" binding:"required"`
	Path string `form:"path" binding:"required"`
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
		Name:        postReq.Name,
		Path:        postReq.Path,
		StorageType: storage.StorageTypeFile,
	}
	if err = bucket.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Re-initialize storage
	storage.Init()
	c.JSON(http.StatusOK, gin.H{"error": ""})
}
