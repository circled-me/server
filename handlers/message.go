package handlers

import (
	"bytes"
	"encoding/json"
	"log"
	"server/db"
	"server/models"
	"server/push"
	"strconv"
	"strings"
	"time"
)

const (
	TypeGroupMessage = "group_message"
	TypeGroupUpdate  = "group_update"
	TypeSeenMessage  = "seen_message"

	GroupUpdateValueNew        = "new"
	GroupUpdateValueLeft       = "left"
	GroupUpdateValueNameChange = "name_change"
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
	Name    string `json:"name"`
}

type SeenMessageDetails struct {
	ID      uint64 `json:"id"`
	GroupID uint64 `json:"group_id"`
	UserID  uint64 `json:"user_id"`
}

type SeenMessage struct {
	Message
	Data SeenMessageDetails `json:"data"`
}

type GroupUpdate struct {
	Message
	Data GroupUpdateDetails `json:"data"`
}

func (sm *SeenMessage) getNotification() *push.Notification {
	return nil
}

func (gm *GroupMessage) getNotification() *push.Notification {
	if gm.Data.ReactionTo > 0 {
		// TODO: Implement reaction notifications - but only to the parent message author
		// title = gm.Data.UserName + " reacted"
		return nil
	}
	body := gm.Data.Content
	if strings.HasPrefix(body, "[image:http") {
		body = "[image]"
	}
	title := gm.Data.UserName
	if len(gm.Data.Group.Name) > 0 {
		title = gm.Data.UserName + " to " + gm.Data.Group.Name
	}
	return &push.Notification{
		Title: title,
		Body:  body,
		Data: map[string]string{
			"type":  TypeGroupMessage,
			"group": strconv.FormatUint(gm.Data.GroupID, 10),
		},
	}
}

func (gu *GroupUpdate) getNotification() *push.Notification {
	if gu.Data.Title == "" {
		return nil
	}
	return &push.Notification{
		Title: gu.Data.Title,
		Body:  gu.Data.Body,
		Data: map[string]string{
			"type":  TypeGroupUpdate,
			"group": strconv.FormatUint(gu.Data.GroupID, 10),
		},
	}
}

func NewSeenMessage(groupID, messageID, userID uint64) (m SeenMessage) {
	m.Message.Type = TypeSeenMessage
	m.Message.Stamp = time.Now().UnixMilli()
	m.Data.ID = messageID
	m.Data.GroupID = groupID
	m.Data.UserID = userID
	return
}

func NewGroupMessage() (m GroupMessage) {
	m.Message.Type = TypeGroupMessage
	m.Message.Stamp = time.Now().UnixMilli()
	return
}

func NewGroupUpdate(groupID uint64, value, title, body, name string) (m GroupUpdate) {
	m.Message.Type = TypeGroupUpdate
	m.Message.Stamp = time.Now().UnixMilli()
	m.Data.GroupID = groupID
	m.Data.Value = value
	m.Data.Title = title
	m.Data.Body = body
	m.Data.Name = name
	return
}

func sendToSocketAndPush(message NotificationGetter, recipients map[uint64]string) {
	buffer := bytes.Buffer{}
	_ = json.NewEncoder(&buffer).Encode(message)
	notification := message.getNotification()
	pushTokens := make([]string, 0, len(recipients))
	for userID, pushToken := range recipients {
		pushTokens = append(pushTokens, pushToken)
		clientID := models.GetUserSocketID(userID)
		connections, exist := connectedUsers.Get(clientID)
		if !exist {
			continue
		}
		// TODO: If initiator, send only confirmation
		for _, conn := range connections {
			if !conn.sendFunc(buffer.Bytes()) {
				conn.removeFrom(clientID)
			}
		}
	}
	// Always send push notification
	if notification != nil {
		notification.SendTo(pushTokens)
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
		recipients := models.GetGroupRecipients(groupMessage.Data.GroupID, user)
		if len(recipients) == 0 {
			log.Printf("User %d does not belong to group %d", user.ID, groupMessage.Data.GroupID)
			return
		}
		groupMessage.saveFor(user)
		groupMessage.propagateToGroupUsers(recipients)

	case TypeSeenMessage:
		seenMessage := SeenMessage{}
		if err := json.Unmarshal(data, &seenMessage); err != nil {
			log.Printf("Not a Seen message: %v", err)
			return
		}
		log.Printf("Seen message: %+v", seenMessage)
		recipients := models.GetGroupRecipients(seenMessage.Data.GroupID, user)
		if len(recipients) == 0 {
			log.Printf("User %d does not belong to group %d", user.ID, seenMessage.Data.GroupID)
			return
		}
		// Update the GroupUser object for the current user and set the SeenMessage field
		err := db.Instance.Exec("update group_users set seen_message = ? where group_id = ? and user_id = ?", seenMessage.Data.ID, seenMessage.Data.GroupID, user.ID).Error
		if err != nil {
			log.Printf("SeenMessage udpate DB error: %v", err)
			return
		}
		sendToSocketAndPush(&seenMessage, recipients)
	}
}

func (message *GroupMessage) saveFor(initiator *models.User) {
	groupMessage := &message.Data
	groupMessage.ServerStamp = time.Now().UnixMilli()
	groupMessage.UserName = initiator.Name
	groupMessage.UserID = initiator.ID
	err := db.Instance.Save(groupMessage).Error
	if err != nil || groupMessage.ID == 0 {
		log.Printf("Couldn't save GroupMessage: %+v, err: %v", *groupMessage, err)
		return
	}
}

func (message *GroupMessage) propagateToGroupUsers(recipients map[uint64]string) {
	groupMessage := &message.Data
	message.Stamp = groupMessage.ServerStamp
	// Just for notification purposes
	if len(recipients) > 2 {
		db.Instance.First(&groupMessage.Group, groupMessage.GroupID)
		if len(groupMessage.Group.Name) == 0 {
			groupMessage.Group.Name = "your group"
		}
	}
	// Propagate
	sendToSocketAndPush(message, recipients)
}
