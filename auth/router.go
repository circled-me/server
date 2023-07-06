package auth

import (
	"net/http"
	"server/models"

	"github.com/gin-gonic/gin"
)

// User is authenticated and posseses the required permissions
type HandlerFunc func(c *gin.Context, user *models.User)

// Router is a wrapper class that adds auth checks + User pre-loading
type Router struct {
	Base *gin.Engine
}

func (cr *Router) baseExec(c *gin.Context, handler HandlerFunc, required []models.Permission) {
	session := LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermissions(required) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	handler(c, &user)
}

func (cr *Router) POST(path string, handler HandlerFunc, required ...models.Permission) {
	cr.Base.POST(path, func(c *gin.Context) {
		cr.baseExec(c, handler, required)
	})
}

func (cr *Router) GET(path string, handler HandlerFunc, required ...models.Permission) {
	cr.Base.GET(path, func(c *gin.Context) {
		cr.baseExec(c, handler, required)
	})
}

func (cr *Router) PUT(path string, handler HandlerFunc, required ...models.Permission) {
	cr.Base.PUT(path, func(c *gin.Context) {
		cr.baseExec(c, handler, required)
	})
}
