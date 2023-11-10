package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"server/db"
	"server/models"
	"server/push"
	"strconv"
	"time"
)

const TypeGroupMessage = "group"

type Message struct {
	Type string `json:"type"`
}

type GroupMessage struct {
	Type  string              `json:"type"`
	Stamp int64               `json:"stamp"`
	Data  models.GroupMessage `json:"data"`
}

func processMessage(user *models.User, data []byte) {
	message := Message{}
	if err := json.Unmarshal(data, &message); err != nil {
		return
	}
	switch message.Type {
	case TypeGroupMessage:
		groupMessage := GroupMessage{}
		if err := json.Unmarshal(data, &groupMessage); err != nil {
			log.Printf("Not a Group message: %v", err)
			return
		}
		log.Printf("Group message: %+v", groupMessage)
		processGroupMessage(user, &groupMessage)
	}
}

func sendMessagePush(pushToken string, groupMessage *models.GroupMessage) {
	// TODO: Create notifications object
	push.Send(&push.Notification{
		UserToken: pushToken,
		Title:     groupMessage.UserName,
		Body:      groupMessage.Content,
		Data: map[string]string{
			"type":  TypeGroupMessage,
			"group": strconv.FormatUint(groupMessage.GroupID, 10),
		},
	})
}

func processGroupMessage(user *models.User, message *GroupMessage) {
	groupMessage := &message.Data
	userIDs := models.LoadGroupUserIDs(groupMessage.GroupID)
	if _, ok := userIDs[user.ID]; !ok {
		log.Printf("User %d does not belong to group %d", user.ID, groupMessage.GroupID)
	}
	groupMessage.ServerStamp = time.Now().UnixMilli()
	groupMessage.UserName = user.Name
	groupMessage.UserID = user.ID
	err := db.Instance.Save(groupMessage).Error
	if err != nil || groupMessage.ID == 0 {
		log.Printf("Couldn't save GroupMessage: %+v, err: %v", *groupMessage, err)
		return
	}
	message.Stamp = groupMessage.ServerStamp
	buffer := bytes.Buffer{}
	_ = json.NewEncoder(&buffer).Encode(*message)

	for userID, pushToken := range userIDs {
		clientID := models.GetUserSocketID(userID)
		connections, exist := connectedUsers.Get(clientID)
		if !exist {
			sendMessagePush(pushToken, groupMessage)
			continue
		}
		sent := false
		for _, conn := range connections {
			if conn.sendFunc(buffer.Bytes()) {
				sent = true
			} else {
				conn.removeFrom(clientID)
			}
		}
		if !sent {
			log.Printf("Couldn't send for user %d", userID)
			sendMessagePush(pushToken, groupMessage)
		}
	}
}
