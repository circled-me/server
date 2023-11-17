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
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type GroupUserInfo struct {
	ID      uint64 `json:"id"`
	Name    string `json:"name"`
	IsAdmin bool   `json:"is_admin"`
}

type GroupInfo struct {
	ID        uint64          `json:"id" form:"id" binding:"required"`
	Name      string          `json:"name" form:"name" binding:"required"`
	Colour    string          `json:"colour" form:"colour"`
	Favourite bool            `json:"favourite" form:"favourite"`
	IsAdmin   bool            `json:"is_admin"`
	Members   []GroupUserInfo `json:"members"`
}

type GroupCreateRequest struct {
	Name string `json:"name" form:"name" binding:"required"`
}

type MessagesRequest struct {
	SinceID uint64 `json:"since_id" form:"since_message"`
}

type GroupDeleteRequest struct {
	ID uint64 `json:"id" form:"id" binding:"required"`
}

func InviteToGroup(c *gin.Context) {
}

func (gi *GroupInfo) loadMembers() {
	rows, err := db.Instance.
		Table("group_users").
		Joins("join `users` on group_users.user_id = `users`.id").
		Select("user_id, name, is_admin").
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
		if err = rows.Scan(&userInfo.ID, &userInfo.Name, &userInfo.IsAdmin); err != nil {
			continue
		}
		gi.Members = append(gi.Members, userInfo)
	}
}

func GroupList(c *gin.Context, user *models.User) {
	rows, err := db.Instance.
		Table("group_users").
		Joins("join `groups` on group_users.group_id = `groups`.id").
		Select("group_id, name, colour, is_favourite, is_admin").
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
		if err = rows.Scan(&groupInfo.ID, &groupInfo.Name, &groupInfo.Colour, &groupInfo.Favourite, &groupInfo.IsAdmin); err != nil {
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
		Name:        r.Name,
		CreatedByID: user.ID,
	}
	result := db.Instance.Create(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// Now create the Group <-> User link
	groupUser := models.GroupUser{
		GroupID: group.ID,
		UserID:  user.ID,
		IsAdmin: true,
	}
	result = db.Instance.Create(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	c.JSON(http.StatusOK, GroupInfo{
		ID:   group.ID,
		Name: group.Name,
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
	if groupUser.IsAdmin || user.HasPermission(models.PermissionAdmin) {
		// We can edit the Group object...
		newMembersMap := map[uint64]bool{}
		for _, m := range r.Members {
			newMembersMap[m.ID] = m.IsAdmin
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
		group.Name = r.Name
		if db.Instance.Omit("Members").Save(&group).Error != nil {
			c.JSON(http.StatusInternalServerError, DBError4Response)
			return
		}
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
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	// Delete the Group object
	group := models.Group{ID: r.ID}
	result := db.Instance.Delete(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}
