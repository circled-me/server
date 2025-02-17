package handlers

import (
	"net/http"
	"server/db"
	"server/models"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4 * 1024,
	WriteBufferSize: 4 * 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type GroupUserInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	IsAdmin     bool   `json:"is_admin"`
	SeenMessage uint64 `json:"seen_message"`
}

type GroupSeenMessage struct {
	GroupID   uint64 `json:"group_id" binding:"required"`
	MessageID uint64 `json:"message_id" binding:"required"`
}

type GroupInfo struct {
	ID          uint64          `json:"id" form:"id" binding:"required"`
	Name        string          `json:"name" form:"name"`
	Colour      string          `json:"colour" form:"colour"`
	Favourite   bool            `json:"favourite" form:"favourite"`
	IsAdmin     bool            `json:"is_admin"`
	Members     []GroupUserInfo `json:"members"`
	SeenMessage uint64          `json:"seen_message"`
}

type GroupCreateRequest struct {
	Members []GroupUserInfo `json:"members" binding:"required"`
}

type MessagesRequest struct {
	SinceID uint64 `json:"since_id" form:"since_message"`
}

type GroupDeleteRequest struct {
	ID uint64 `json:"id" binding:"required"`
}

func InviteToGroup(c *gin.Context) {
}

func (gi *GroupInfo) loadMembers() {
	rows, err := db.Instance.
		Table("group_users").
		Joins("join `users` on group_users.user_id = `users`.id").
		Select("user_id, name, is_admin, seen_message").
		Where("group_id = ?", gi.ID).
		Order("group_users.created_at").
		Rows()

	if err != nil {
		return
	}
	defer rows.Close()
	gi.Members = []GroupUserInfo{}
	for rows.Next() {
		userInfo := GroupUserInfo{}
		if err = rows.Scan(&userInfo.ID, &userInfo.Name, &userInfo.IsAdmin, &userInfo.SeenMessage); err != nil {
			continue
		}
		gi.Members = append(gi.Members, userInfo)
	}
}

func GroupList(c *gin.Context, user *models.User) {
	rows, err := db.Instance.
		Table("group_users").
		Joins("join `groups` on group_users.group_id = `groups`.id").
		Select("group_id, name, colour, is_favourite, is_admin, seen_message").
		Where("user_id = ?", user.ID).
		Order("is_favourite DESC, `groups`.updated_at DESC").
		Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := []GroupInfo{}
	isGlobalAdmin := user.HasPermission(models.PermissionAdmin)
	for rows.Next() {
		groupInfo := GroupInfo{}
		if err = rows.Scan(&groupInfo.ID, &groupInfo.Name, &groupInfo.Colour, &groupInfo.Favourite, &groupInfo.IsAdmin, &groupInfo.SeenMessage); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		if isGlobalAdmin {
			groupInfo.IsAdmin = true
		}
		groupInfo.loadMembers()
		result = append(result, groupInfo)
	}
	c.JSON(http.StatusOK, result)
}

// GroupCreate creates a Group object and also a GroupUser for the current user
func GroupCreate(c *gin.Context, user *models.User) {
	if !user.HasPermission(models.PermissionAdmin) && !user.HasPermission(models.PermissionCanCreateGroups) {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	r := GroupCreateRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	group := models.Group{
		CreatedByID: user.ID,
	}
	result := db.Instance.Create(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// Now create the Group <-> User link
	for _, gu := range r.Members {
		groupUser := models.GroupUser{
			GroupID: group.ID,
			UserID:  gu.ID,
			IsAdmin: gu.IsAdmin,
		}
		result = db.Instance.Create(&groupUser)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
	}
	// Send notification to all members
	newMembersMessage := NewGroupUpdate(group.ID, GroupUpdateValueNew, user.Name+" added you to a new group", "Check your groups and start chatting", "")
	members := models.LoadGroupUserIDs(group.ID)
	delete(members, user.ID)
	go sendToSocketAndPush(&newMembersMessage, members)

	c.JSON(http.StatusOK, GroupInfo{
		ID:      group.ID,
		Members: r.Members,
		IsAdmin: true,
	})
}

// GroupSave updates the Group and GroupUser objects for the current user
func GroupSave(c *gin.Context, user *models.User) {
	r := GroupInfo{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	// Load the GroupUser object
	groupUser := models.GroupUser{
		GroupID: r.ID,
		UserID:  user.ID,
	}
	result := db.Instance.First(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// Update the fields
	groupUser.Colour = r.Colour
	groupUser.IsFavourite = r.Favourite
	result = db.Instance.Save(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	// Load the Group object
	group := models.Group{ID: r.ID}
	result = db.Instance.Preload("Members").First(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError3Response)
		return
	}
	lastMembersTokenMap := models.LoadGroupUserIDs(group.ID)
	sameMembers := len(lastMembersTokenMap) == len(r.Members)
	if sameMembers {
		for _, member := range r.Members {
			if _, present := lastMembersTokenMap[member.ID]; !present {
				sameMembers = false
			}
		}
	}
	// Update name
	if (groupUser.IsAdmin || user.HasPermission(models.PermissionAdmin)) && group.Name != r.Name {
		group.Name = r.Name
		if db.Instance.Omit("Members").Save(&group).Error != nil {
			c.JSON(http.StatusInternalServerError, DBError4Response)
			return
		}
		// Send notification to current members
		updateMessage := NewGroupUpdate(group.ID, GroupUpdateValueNameChange, user.Name+" changed the group name to '"+r.Name+"'", "Check '"+r.Name+"'", r.Name)
		go sendToSocketAndPush(&updateMessage, lastMembersTokenMap)
	}
	// Update members?
	if (groupUser.IsAdmin || user.HasPermission(models.PermissionAdmin)) && !sameMembers {
		// We can edit the Group object...
		newMembersMap := map[uint64]bool{}
		for _, m := range r.Members {
			newMembersMap[m.ID] = m.IsAdmin
			delete(lastMembersTokenMap, m.ID)
		}
		oldMembersMap := map[uint64]bool{}
		// Modify the current members
		for _, member := range group.Members {
			if isAdmin, ok := newMembersMap[member.UserID]; ok {
				// Just update the old GroupUser objects as they contain preferences
				member.IsAdmin = isAdmin
				db.Instance.Save(&member)
			} else {
				// Remove deleted ones
				db.Instance.Delete(&member)
			}
			oldMembersMap[member.UserID] = member.IsAdmin
		}
		// Add new members
		for _, m := range r.Members {
			if _, ok := oldMembersMap[m.ID]; ok {
				continue
			}
			db.Instance.Save(&models.GroupUser{
				GroupID: group.ID,
				UserID:  m.ID,
				IsAdmin: m.IsAdmin,
			})
		}
		if db.Instance.Omit("Members").Save(&group).Error != nil {
			c.JSON(http.StatusInternalServerError, DBError4Response)
			return
		}
		// Send notification to new members
		newMembersTokenMap := models.LoadGroupUserIDs(group.ID)
		for k := range newMembersTokenMap {
			if _, exists := newMembersMap[k]; !exists {
				delete(newMembersTokenMap, k)
			}
		}
		newMembersMessage := NewGroupUpdate(group.ID, GroupUpdateValueNew, user.Name+" added you to a new chat", "Check your groups and start chatting", "")
		go sendToSocketAndPush(&newMembersMessage, newMembersTokenMap)
		// Send notification to retired members
		retiredMembersMessage := NewGroupUpdate(group.ID, GroupUpdateValueLeft, "", "", "")
		go sendToSocketAndPush(&retiredMembersMessage, lastMembersTokenMap)
	}
	c.JSON(http.StatusOK, GroupInfo{
		ID:        group.ID,
		Name:      group.Name,
		Colour:    groupUser.Colour,
		Favourite: groupUser.IsFavourite,
		IsAdmin:   groupUser.IsAdmin,
	})
}

// GroupDelete deletes the Group and all of its dependants (via foreign keys)
func GroupDelete(c *gin.Context, user *models.User) {
	r := GroupDeleteRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	// Load the GroupUser object
	groupUser := models.GroupUser{
		GroupID: r.ID,
		UserID:  user.ID,
	}
	if db.Instance.First(&groupUser).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if !groupUser.IsAdmin && !user.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	members := models.LoadGroupUserIDs(r.ID)
	delete(members, user.ID)
	leftMessage := NewGroupUpdate(r.ID, GroupUpdateValueLeft, "", "", "")

	// Delete the Group object
	group := models.Group{ID: r.ID}
	result := db.Instance.Delete(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// Send WS message to ex-members
	go sendToSocketAndPush(&leftMessage, members)

	c.JSON(http.StatusOK, OKResponse)
}
