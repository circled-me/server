package handlers

import (
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type UserCreateRequest struct {
	Name     string `form:"name" binding:"required"`
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
}
type UserLoginRequest struct {
	Email    string `form:"email" binding:"required"`
	Password string `form:"password" binding:"required"`
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
		"name":        user.Name,
		"user_id":     user.ID,
		"permissions": permissions,
	})
}

func UserCreate(c *gin.Context) {
	// postReq := UserCreateRequest{}
	// err := c.ShouldBindWith(&postReq, binding.Form)
	// if err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }
	// user, err := models.UserCreate(postReq.Name, postReq.Email, postReq.Password)
	// if err != nil {
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	// c.JSON(http.StatusOK, gin.H{"error": "", "user": user})
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
