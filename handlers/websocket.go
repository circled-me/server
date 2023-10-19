package handlers

import (
	"log"
	"server/models"

	"github.com/gin-gonic/gin"
)

func WebSocket(c *gin.Context, user *models.User) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer conn.Close()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)
		if string(message) == "ping" {
			conn.WriteMessage(mt, []byte("pong"))
		}
		if string(message) == "pong" {
			continue
		}
		err = conn.WriteMessage(mt, message)
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}
