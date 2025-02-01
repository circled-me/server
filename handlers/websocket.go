package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"server/db"
	"server/models"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
	cmap "github.com/orcaman/concurrent-map/v2"
)

// sendSocketFunc returns true if data was successfully sent
type sendSocketFunc func([]byte) bool
type connectedClient struct {
	sendFunc sendSocketFunc
}

// connectedClients is needed as a user may be connected more than once
type connectedClients []*connectedClient

var (
	connectedUsers = cmap.New[connectedClients]()
)

func (c *connectedClient) addTo(id string) {
	connectedUsers.Upsert(id, connectedClients{c}, func(exist bool, valueInMap, newValue connectedClients) connectedClients {
		if exist {
			return append(valueInMap, c)
		}
		return newValue
	})
}

func (c *connectedClient) removeFrom(id string) {
	connectedUsers.Upsert(id, connectedClients{}, func(exist bool, valueInMap, newValue connectedClients) connectedClients {
		if !exist {
			// TODO: Cleanup this empty arrays
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

func withMessagesFor(user *models.User, since int64, callback func(models.GroupMessage)) {
	rows, err := db.Instance.
		Table("group_messages").
		Select("group_messages.id, group_messages.group_id, server_stamp, client_stamp, "+
			"users.id, users.name, content, reply_to, group_concat(group_message_reactions.user_id||':'||group_message_reactions.reaction)").
		Joins("join group_users ON group_users.user_id = ? AND group_users.group_id = group_messages.group_id", user.ID).
		Joins("join users ON users.id = group_messages.user_id").
		Joins("left join group_message_reactions ON group_message_reactions.id = group_messages.id").
		Where("group_messages.id > ?", since).
		Group("group_messages.id").
		Order("group_messages.id ASC").
		Rows()
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		groupMessage := models.GroupMessage{}
		reactions := ""
		reactionsPtr := &reactions
		if err := rows.Scan(&groupMessage.ID, &groupMessage.GroupID, &groupMessage.ServerStamp, &groupMessage.ClientStamp,
			&groupMessage.UserID, &groupMessage.UserName, &groupMessage.Content, &groupMessage.ReplyTo, &reactionsPtr); err != nil {

			log.Printf("DB error: %v", err)
			continue
		}
		groupMessage.Reactions = []models.GroupMessageReaction{}
		for _, reaction := range strings.Split(reactions, ",") {
			parts := strings.Split(reaction, ":")
			if len(parts) != 2 {
				continue
			}
			uID, _ := strconv.ParseUint(parts[0], 10, 64)
			groupMessage.Reactions = append(groupMessage.Reactions, models.GroupMessageReaction{
				UserID:   uID,
				Reaction: parts[1],
			})
		}
		callback(groupMessage)
	}
}

func WebSocket(c *gin.Context, user *models.User) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer conn.Close()

	// Setup client
	isConnected := true
	id := models.GetUserSocketID(user.ID)
	log.Printf("websocket connected, id: %s", id)
	client := connectedClient{}
	client.sendFunc = func(data []byte) bool {
		if !isConnected {
			return false
		}
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			log.Println("write err:", err)
			isConnected = false
			client.removeFrom(id)
			return false
		}
		return true
	}
	r := MessagesRequest{}
	if err = c.ShouldBindWith(&r, binding.Form); err == nil {
		message := NewGroupMessage()
		withMessagesFor(user, int64(r.SinceID), func(groupMessage models.GroupMessage) {
			message.Data = groupMessage
			message.Stamp = groupMessage.ServerStamp
			buffer := bytes.Buffer{}
			_ = json.NewEncoder(&buffer).Encode(message)
			client.sendFunc(buffer.Bytes())
		})
	}
	client.addTo(id)
	defer client.removeFrom(id)
	// Main read cycle
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("read err:", err)
			isConnected = false
			break
		}
		if string(message) == "ping" {
			conn.WriteMessage(mt, []byte("pong"))
		}
		if string(message) == "pong" {
			continue
		}
		// log.Printf("recv: %s", message)
		processMessage(user, message)
	}
}
