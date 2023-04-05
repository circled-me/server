package auth

import (
	"server/db"
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

func (s *Session) User() (user models.User) {
	id := s.Get("id")
	if id == nil {
		return
	}
	user.ID = id.(uint64)
	if db.Instance.Preload("Grants").Preload("Bucket").First(&user).Error != nil {
		user.ID = 0
	}
	return
}
