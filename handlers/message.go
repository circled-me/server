package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"server/db"
	"server/models"
	"slices"
	"time"
)

type Message struct {
	Type string `json:"type"`
}

type GroupMessage struct {
	Type string              `json:"type"`
	Data models.GroupMessage `json:"data"`
}

func processMessage(user *models.User, data []byte) {
	message := Message{}
	if err := json.Unmarshal(data, &message); err != nil {
		return
	}
	switch message.Type {
	case "group":
		groupMessage := GroupMessage{}
		if err := json.Unmarshal(data, &groupMessage); err != nil {
			log.Printf("Not a Group message: %v", err)
			return
		}
		log.Printf("Group message: %+v", groupMessage)
		processGroupMessage(user, &groupMessage)
	}
}

func processGroupMessage(user *models.User, message *GroupMessage) {
	groupMessage := &message.Data
	userIDs := models.LoadGroupUserIDs(groupMessage.GroupID)
	if !slices.Contains(userIDs, user.ID) {
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
	buffer := bytes.Buffer{}
	_ = json.NewEncoder(&buffer).Encode(*message)

	for _, userID := range userIDs {
		connections, exist := ConnectedUsers.Get(models.GetUserSocketID(userID))
		if !exist {
			// TODO: Send as notification
			continue
		}
		sent := false
		for _, conn := range connections {
			if conn.fun(buffer.Bytes()) {
				sent = true
			}
		}
		if !sent {
			// TODO: Send as notification
		}
	}
}
