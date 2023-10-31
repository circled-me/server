package handlers

import (
	"log"
	"server/models"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// SendSocketFunc returns true if data was successfully sent
type SendSocketFunc func([]byte) bool
type ConnectedClient struct {
	fun SendSocketFunc
}

// ConnectedClients is needed as a user may be connected more than once
type ConnectedClients []*ConnectedClient

var (
	ConnectedUsers = cmap.New[ConnectedClients]()
)

func addClient(id string, c *ConnectedClient) {
	ConnectedUsers.Upsert(id, ConnectedClients{c}, func(exist bool, valueInMap, newValue ConnectedClients) ConnectedClients {
		if exist {
			return append(valueInMap, c)
		}
		return newValue
	})
}

func removeClient(id string, c *ConnectedClient) {
	ConnectedUsers.Upsert(id, ConnectedClients{}, func(exist bool, valueInMap, newValue ConnectedClients) ConnectedClients {
		if !exist {
			return newValue
		}
		for _, oc := range valueInMap {
			if oc == c {
				continue
			}
			newValue = append(newValue, oc)
		}
		return newValue
	})
}

func WebSocket(c *gin.Context, user *models.User) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()

	// Setup client
	isConnected := true
	id := models.GetUserSocketID(user.ID)
	client := ConnectedClient{}
	client.fun = func(data []byte) bool {
		if !isConnected {
			return false
		}
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("write err:", err)
			isConnected = false
			return false
		}
		return true
	}
	addClient(id, &client)
	defer removeClient(id, &client)
	// Main read cycle
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			isConnected = false
			break
		}
		log.Printf("recv: %s", message)
		if string(message) == "ping" {
			conn.WriteMessage(mt, []byte("pong"))
		}
		if string(message) == "pong" {
			continue
		}
		processMessage(user, message)
	}
}
