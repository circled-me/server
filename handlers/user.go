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
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

func UserLogin(c *gin.Context) {
	postReq := UserLoginRequest{}
	err := c.ShouldBindWith(&postReq, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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
	c.JSON(http.StatusOK, gin.H{"error": "", "name": user.Name, "permissions": permissions})
}

func UserCreate(c *gin.Context) {
	postReq := UserCreateRequest{}
	err := c.ShouldBindWith(&postReq, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	user, err := models.UserCreate(postReq.Name, postReq.Email, postReq.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": "", "user": user})
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
	rows, err := db.Instance.Table("users").Select("id, name").Where("id != ?", user.ID).Order("created_at DESC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []UserInfo{}
	for rows.Next() {
		userInfo := UserInfo{}
		if err = rows.Scan(&userInfo.ID, &userInfo.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, userInfo)
	}
	c.JSON(http.StatusOK, result)
}
