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

const (
	TypeGroupMessage = "group_message"
	TypeGroupUpdate  = "group_update"

	GroupUpdateValueNew  = "new"
	GroupUpdateValueLeft = "left"
)

type NotificationGetter interface {
	getNotification() *push.Notification
}

type Message struct {
	Type  string `json:"type"`
	Stamp int64  `json:"stamp"`
}

type GroupMessage struct {
	Message
	Data models.GroupMessage `json:"data"`
}

type GroupUpdateDetails struct {
	GroupID uint64 `json:"group_id"`
	Value   string `json:"value"`
	Title   string `json:"title"`
	Body    string `json:"body"`
}

type GroupUpdate struct {
	Message
	Data GroupUpdateDetails `json:"data"`
}

func (gm *GroupMessage) getNotification() *push.Notification {
	return &push.Notification{
		Title: gm.Data.UserName,
		Body:  gm.Data.Content,
		Data: map[string]string{
			"type":  TypeGroupMessage,
			"group": strconv.FormatUint(gm.Data.GroupID, 10),
		},
	}
}

func (sm *GroupUpdate) getNotification() *push.Notification {
	if sm.Data.Title == "" {
		return nil
	}
	return &push.Notification{
		Title: sm.Data.Title,
		Body:  sm.Data.Body,
		// TODO: Data
	}
}

func NewGroupMessage() (m GroupMessage) {
	m.Message.Type = TypeGroupMessage
	m.Message.Stamp = time.Now().UnixMilli()
	return
}

func NewGroupUpdate(groupID uint64, value, title, body string) (m GroupUpdate) {
	m.Message.Type = TypeGroupUpdate
	m.Message.Stamp = time.Now().UnixMilli()
	m.Data.GroupID = groupID
	m.Data.Value = value
	m.Data.Title = title
	m.Data.Body = body
	return
}

func sendToSocketOrPush(message NotificationGetter, recipients map[uint64]string) {
	buffer := bytes.Buffer{}
	_ = json.NewEncoder(&buffer).Encode(message)
	notification := message.getNotification()

	for userID, pushToken := range recipients {
		clientID := models.GetUserSocketID(userID)
		connections, exist := connectedUsers.Get(clientID)
		if !exist {
			if notification != nil {
				notification.SendTo(pushToken)
			}
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
		if !sent && notification != nil {
			notification.SendTo(pushToken)
		}
	}
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
		groupMessage.saveAndPropagate(user)
	}
}

func (message *GroupMessage) saveAndPropagate(initiator *models.User) {
	groupMessage := &message.Data
	recipients := models.LoadGroupUserIDs(groupMessage.GroupID)
	if _, ok := recipients[initiator.ID]; !ok {
		log.Printf("User %d does not belong to group %d", initiator.ID, groupMessage.GroupID)
		return
	}
	groupMessage.ServerStamp = time.Now().UnixMilli()
	groupMessage.UserName = initiator.Name
	groupMessage.UserID = initiator.ID
	err := db.Instance.Save(groupMessage).Error
	if err != nil || groupMessage.ID == 0 {
		log.Printf("Couldn't save GroupMessage: %+v, err: %v", *groupMessage, err)
		return
	}
	message.Stamp = groupMessage.ServerStamp
	// Propagate
	sendToSocketOrPush(message, recipients)
}
