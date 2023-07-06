package handlers

import (
	"log"
	"net/http"
	"server/db"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

func UploadShare(c *gin.Context, user *models.User) {
	shareInfo := models.NewUploadRequest(user.ID)
	result := db.Instance.Create(&shareInfo)
	if result.Error != nil {
		log.Print(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": "/w/upload/" + shareInfo.Token + "/"})
}
