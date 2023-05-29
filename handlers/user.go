package handlers

import (
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type UserLoginRequest struct {
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
	Token    string `form:"token"`
}

type UserInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	Permissions []int  `json:"permissions"`
	Bucket      uint64 `json:"bucket"`
}

func UserLogin(c *gin.Context) {
	postReq := UserLoginRequest{}
	err := c.ShouldBindWith(&postReq, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// New user has been invited
	if postReq.Token != "" {
		invite := models.Invitation{
			Token: postReq.Token,
		}
		if err = db.Instance.Find(&invite).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no such token"})
			return
		}
		user := models.User{ID: invite.UserID}
		if err = db.Instance.Find(&user).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "no such user"})
			return
		}
		user.Email = postReq.Email
		if err = models.UserSetPassword(user, postReq.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save user"})
			return
		}
		// Not needed any more...
		_ = db.Instance.Delete(&invite)
	}
	// TODO: change this
	// Check if we have a brand new instance
	var count int64
	result := db.Instance.Raw("select 1 where exists(select 1 from users)").Scan(&count)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	if count == 0 {
		// No users exist - create one with the provided details (name defaults to email)
		user, err := models.UserCreate(postReq.Email, postReq.Email, postReq.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		err = db.Instance.Save(&models.Grant{
			GrantorID:  user.ID,
			UserID:     user.ID,
			Permission: models.PermissionAdmin,
		}).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	// Proceed with standard login
	user, err := models.UserLogin(postReq.Email, postReq.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	permissions := user.GetPermissions()
	session := auth.LoadSession(c)
	session.Set("id", user.ID)
	session.Set("permissions", permissions)
	session.Save()
	c.JSON(http.StatusOK, gin.H{
		"error":       "",
		"name":        user.Email,
		"user_id":     user.ID,
		"permissions": permissions,
	})
}

func UserSave(c *gin.Context) {
	session := auth.LoadSession(c)
	currentUser := session.User()
	if currentUser.ID == 0 || !currentUser.HasPermission(models.PermissionAdmin) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	req := UserInfo{}
	err := c.ShouldBindWith(&req, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Bucket == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "select storage bucket"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty name"})
		return
	}
	user := models.User{ID: req.ID}
	if user.ID > 0 {
		if err = db.Instance.Preload("Grants").Find(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		// New user
		user.Email = req.Email
	}
	user.BucketID = &req.Bucket
	user.Name = req.Name
	for _, g := range user.Grants {
		db.Instance.Delete(&g)
	}
	user.Grants = []models.Grant{}
	for _, p := range req.Permissions {
		user.Grants = append(user.Grants, models.Grant{
			GrantorID:  currentUser.ID,
			Permission: models.Permission(p),
		})
	}
	if err = db.Instance.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	token := ""
	if req.ID == 0 {
		// This was a new user - create invitation token
		invite := models.NewInvitation(user.ID)
		if err = db.Instance.Save(&invite).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		token = invite.Token
	}
	c.JSON(http.StatusOK, gin.H{
		"error": "",
		"token": token,
	})
}

func UserGetPermissions(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "name": "", "permissions": []int{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": "", "name": user.Name, "permissions": user.GetPermissions()})
}

func UserList(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Not Found", "name": "", "permissions": []int{}})
		return
	}
	users := []models.User{}
	err := db.Instance.Preload("Grants").Order("created_at ASC").Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	result := []UserInfo{}
	for _, u := range users {
		userInfo := UserInfo{
			ID:          u.ID,
			Name:        u.Name,
			Email:       u.Email,
			Bucket:      *u.BucketID,
			Permissions: u.GetPermissions(),
		}
		result = append(result, userInfo)
	}
	c.JSON(http.StatusOK, result)
}
