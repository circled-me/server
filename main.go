package main

import (
	"server/db"
	"server/handlers"
	"server/models"
	"server/storage"

	"github.com/gin-contrib/sessions"
	gormsessions "github.com/gin-contrib/sessions/gorm"
	"github.com/gin-gonic/gin"
)

const (
	sessionStoreKey       = "this is a long key" // TODO: convert to env variable
	sessionCookieName     = "token"
	sessionExpirationTime = 365 * 86400 // 1 year
)

func main() {
	db.Init()
	models.Init()
	storage.Init()

	router := gin.Default()
	router.SetTrustedProxies([]string{})

	cookieStore := gormsessions.NewStore(db.Instance, true, []byte(sessionStoreKey))
	// cookieStore := memstore.NewStore([]byte(sessionStoreKey))
	cookieStore.Options(sessions.Options{MaxAge: sessionExpirationTime})
	router.Use(sessions.Sessions(sessionCookieName, cookieStore))

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	router.GET("/incr", func(c *gin.Context) {
		session := sessions.Default(c)
		var count int
		v := session.Get("count")
		if v == nil {
			count = 0
		} else {
			count = v.(int)
			count++
		}
		session.Set("count", count)
		session.Save()
		c.JSON(200, gin.H{"count": count})
	})
	router.GET("/get", func(c *gin.Context) {
		session := sessions.Default(c)
		c.JSON(200, gin.H{"count": session.Get("count")})
	})
	router.POST("/backup/check", handlers.BackupCheck)
	router.POST("/backup/do", handlers.BackupAsset)
	router.POST("/bucket/create", handlers.BucketCreate)
	router.POST("/user/create", handlers.UserCreate)
	router.POST("/user/login", handlers.UserLogin)
	router.Run("0.0.0.0:8080")
}
