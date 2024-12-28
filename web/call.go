package web

import (
	"net/http"
	"server/models"

	"github.com/gin-gonic/gin"
)

func CallView(c *gin.Context) {
	id := c.Param("token")
	vc, err := models.VideoCallByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	if vc.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}
	c.HTML(http.StatusOK, "call_view.tmpl", gin.H{
		"token": vc.ID,
	})
}
