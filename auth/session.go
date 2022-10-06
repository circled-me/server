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

func (s *Session) GetPermissions() []int {
	permissions := s.Get("permissions")
	if permissions == nil {
		return []int{}
	}
	return permissions.([]int)
}

func (s *Session) HasPermission(required models.Permission) bool {
	for _, permission := range s.GetPermissions() {
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
