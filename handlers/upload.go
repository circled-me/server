package handlers

import (
	"log"
	"net/http"
	"server/db"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

type UploadShareResponse struct {
	Path string `json:"path"`
}

func UploadShare(c *gin.Context, user *models.User) {
	shareInfo := models.NewUploadRequest(user.ID)
	result := db.Instance.Create(&shareInfo)
	if result.Error != nil {
		log.Print(result.Error)
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, UploadShareResponse{"/w/upload/" + shareInfo.Token + "/"})
}
