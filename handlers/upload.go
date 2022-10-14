package handlers

import (
	"log"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

func UploadShare(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	shareInfo := models.NewUploadRequest(user.ID)
	result := db.Instance.Create(&shareInfo)
	if result.Error != nil {
		log.Print(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": "/w/upload/" + shareInfo.Token + "/"})
}
