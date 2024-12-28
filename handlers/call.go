package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"server/utils"
	"server/webrtc"

	"github.com/gin-gonic/gin"
)

func CallLink(c *gin.Context, user *models.User) {
	if !user.HasPermission(models.PermissionAdmin) && !user.HasPermission(models.PermissionCanCreateGroups) {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	vc, err := models.VideoCallForUser(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, NopeResponse)
		return
	}
	if c.Query("reset") == "1" {
		vc.ID = utils.Rand8BytesToBase62()
		if err = db.Instance.Save(&vc).Error; err != nil {
			c.JSON(http.StatusInternalServerError, NopeResponse)
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"path": "/call/" + vc.ID})
}

func CallWebSocket(c *gin.Context) {
	token := c.Param("token")
	_, err := models.VideoCallByID(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer conn.Close()

	room := webrtc.GetRoom(token)
	// Wait for the client to send their ID
	_, id, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}
	client, isNew := room.SetUpClient(conn, string(id))
	if isNew {
		room.MessageTo(client.ID, map[string]interface{}{
			"type": "id",
			"id":   client.ID,
		})
		log.Printf("Client %s joined\n", client.ID)
	} else {
		log.Printf("Client %s reconnected\n", client.ID)
	}
	room.Broadcast(client.ID, map[string]interface{}{
		"type": "joined",
		"from": client.ID,
	})
	removeClient := func() {
		room.Broadcast(client.ID, map[string]interface{}{
			"type": "left",
			"from": client.ID,
		})
		room.RemoveClient(client)
	}
	// Start client message loop
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			removeClient()
			break
		}
		log.Println("Got message:" + string(message))

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Invalid JSON message: %s\n", string(message))
			continue
		}
		switch msg["type"] {
		case "ping":
			room.SeenClient(client)
		case "leave":
			removeClient()
			log.Printf("Client %s left\n", client.ID)
		case "offer":
			fallthrough
		case "candidate":
			fallthrough
		case "answer":
			msg["from"] = client.ID
			recipient, exists := msg["to"].(string)
			if !exists || recipient == "" {
				log.Printf("Invalid message: %v\n", msg)
				continue
			}
			room.MessageTo(recipient, msg)
		}
	}
}
