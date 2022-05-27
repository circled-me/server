package handlers

import (
	"net/http"
	"server/auth"
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
	permissions := make([]int, 0)
	for _, grant := range user.Grants {
		permissions = append(permissions, int(grant.Permission))
	}
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
