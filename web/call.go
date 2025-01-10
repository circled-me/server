package web

import (
	"log"
	"net/http"
	"server/auth"
	"server/config"
	"server/models"

	"github.com/gin-gonic/gin"
)

func CallView(c *gin.Context) {
	id := c.Param("id")
	vc, err := models.VideoCallByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
		return
	}
	if vc.ID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
		return
	}
	// Load the user session by setting the token cookie manually from the query
	sessionToken := c.Query("token")
	c.Request.Header.Add("Cookie", "token="+sessionToken)
	session := auth.LoadSession(c)
	user := session.User()
	log.Printf("User %d is trying to join call %s", user.ID, vc.ID)

	c.HTML(http.StatusOK, "call_view.tmpl", gin.H{
		"id":       vc.ID,
		"wsQuery":  "token=" + sessionToken,
		"turnIP":   config.TURN_SERVER_IP,
		"turnPort": config.TURN_SERVER_PORT,
	})
}
