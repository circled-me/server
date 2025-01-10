package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/push"
	"server/utils"
	"server/webrtc"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
		newID := utils.Rand8BytesToBase62()
		if err = db.Instance.Exec("update video_calls set id = ? where id = ?", newID, vc.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, NopeResponse)
			return
		}
		vc.ID = newID
	}
	c.JSON(http.StatusOK, gin.H{"path": "/call/" + vc.ID})
}

func sendCallNotificationTo(users map[uint64]string, from *models.User, callURL string) {
	log.Printf("Sending call notification to %v, url: %s\n", users, callURL)
	pushTokens := make([]string, 0, len(users))
	for userID, pushToken := range users {
		if userID == from.ID {
			continue
		}
		pushTokens = append(pushTokens, pushToken)
	}
	if len(pushTokens) == 0 {
		return
	}
	callerName := from.Name
	if from.ID == 0 || from.Name == "" {
		callerName = "<Unknown>"
	}
	uuid := uuid.New()
	notification := &push.Notification{
		Type: push.NotificationTypeCall,
		Data: map[string]string{
			"id":          uuid.String(),
			"caller_name": callerName,
			"caller_id":   callURL,
		},
	}
	if err := notification.SendTo(pushTokens); err != nil {
		log.Printf("Failed to send call notification to %v, error: %v", pushTokens, err)
	}
}

func CallWebSocket(c *gin.Context) {
	stringID := c.Param("id")
	vc, err := models.VideoCallByID(stringID)
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

	// Load the user session by setting the token cookie manually from the query
	c.Request.Header.Add("Cookie", "token="+c.Query("token"))
	session := auth.LoadSession(c)
	user := session.User()
	log.Printf("User %d is trying to connect to WS %s", user.ID, vc.ID)

	room, isNewRoom := webrtc.GetRoom(stringID)
	// Wait for the client to send their ID
	_, id, err := conn.ReadMessage()
	if err != nil {
		log.Println(err)
		return
	}
	client, isNewClient, numClients := room.SetUpClient(conn, string(id), user.ID)
	if isNewClient {
		room.MessageTo(client.ID, map[string]interface{}{
			"type": "id",
			"id":   client.ID,
		})
		log.Printf("Client %s joined\n", client.ID)
	} else {
		log.Printf("Client %s reconnected\n", client.ID)
	}
	if isNewRoom || (isNewClient && numClients == 1) {
		// Send call notification to all users that are not in the call
		sendCallNotificationTo(vc.GetOwners(), &user, c.Query("url"))
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
