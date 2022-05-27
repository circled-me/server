package auth

import (
	"server/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

type Session struct {
	sessions.Session
}

func LoadSession(c *gin.Context) *Session {
	return &Session{
		Session: sessions.Default(c),
	}
}

func (s *Session) HasPermission(required models.Permission) bool {
	permissions := s.Get("permissions")
	if permissions == nil {
		return false
	}
	for _, permission := range permissions.([]int) {
		if permission == int(required) {
			return true
		}
	}
	return false
}

func (s *Session) UserID() uint64 {
	id := s.Get("id")
	if id == nil {
		return 0
	}
	return id.(uint64)
}
