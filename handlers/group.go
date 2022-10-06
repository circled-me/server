package handlers

import (
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type GroupUserInfo struct {
	ID        uint64 `json:"id"`
	Name      string `json:"name"`
	IsAdmin   bool   `json:"is_admin"`
	CanInvite bool   `json:"can_invite"`
}

type GroupInfo struct {
	ID        uint64          `json:"id" form:"id" binding:"required"`
	Name      string          `json:"name" form:"name" binding:"required"`
	Colour    string          `json:"colour" form:"colour"`
	Favourite bool            `json:"favourite" form:"favourite"`
	IsAdmin   bool            `json:"is_admin"`
	CanInvite bool            `json:"can_invite"`
	Members   []GroupUserInfo `json:"members"`
}

type GroupCreateRequest struct {
	Name string `form:"name" binding:"required"`
}

type GroupDeleteRequest struct {
	ID uint64 `json:"id" form:"id" binding:"required"`
}

func InviteToGroup(c *gin.Context) {
}

func GroupList(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.
		Table("group_users").
		Joins("join `groups` on group_users.group_id = `groups`.id").
		Select("group_id, name, colour, is_favourite, can_invite, is_admin").Where("user_id = ?", userID).
		Order("is_favourite DESC, `groups`.updated_at DESC").Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []GroupInfo{}
	for rows.Next() {
		groupInfo := GroupInfo{}
		if err = rows.Scan(&groupInfo.ID, &groupInfo.Name, &groupInfo.Colour, &groupInfo.Favourite, &groupInfo.CanInvite, &groupInfo.IsAdmin); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		// TODO: Members
		result = append(result, groupInfo)
	}
	c.JSON(http.StatusOK, result)
}

// GroupCreate creates a Group object and also a GroupUser one for the current user
func GroupCreate(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || (!session.HasPermission(models.PermissionAdmin) && !session.HasPermission(models.PermissionCanCreateGroups)) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := GroupCreateRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	group := models.Group{
		Name:        r.Name,
		CreatedByID: userID,
	}
	result := db.Instance.Create(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	// Now create the Group <-> User link
	groupUser := models.GroupUser{
		GroupID:   group.ID,
		UserID:    userID,
		CanInvite: true,
		IsAdmin:   true,
	}
	result = db.Instance.Create(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, GroupInfo{
		ID:   group.ID,
		Name: group.Name,
	})
}

// GroupSave updates the Group and GroupUser objects for the current user
func GroupSave(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := GroupInfo{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Load the GroupUser object
	groupUser := models.GroupUser{
		GroupID: r.ID,
		UserID:  userID,
	}
	result := db.Instance.First(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	// Update the fields
	groupUser.Colour = r.Colour
	groupUser.IsFavourite = r.Favourite
	result = db.Instance.Save(&groupUser)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	// Load the Group object
	group := models.Group{ID: r.ID}
	result = db.Instance.Find(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if (groupUser.IsAdmin || session.HasPermission(models.PermissionAdmin)) && group.Name != r.Name {
		// We can edit the Group object
		group.Name = r.Name
		if result = db.Instance.Save(&group); result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
	}
	c.JSON(http.StatusOK, GroupInfo{
		ID:        group.ID,
		Name:      group.Name,
		Colour:    groupUser.Colour,
		Favourite: groupUser.IsFavourite,
		IsAdmin:   groupUser.IsAdmin,
		CanInvite: groupUser.CanInvite,
	})
}

// GroupDelete deletes the Group and all of its dependants (via foreign keys)
func GroupDelete(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := GroupDeleteRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Delete the Group object
	group := models.Group{ID: r.ID}
	result := db.Instance.Delete(&group)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, "DELETED")
}
