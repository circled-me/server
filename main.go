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
	db.Init(GetMySQLDSN())
	models.Init()
	storage.Init()

	router := gin.Default()
	router.SetTrustedProxies([]string{})

	cookieStore := gormsessions.NewStore(db.Instance, true, []byte(sessionStoreKey))
	cookieStore.Options(sessions.Options{MaxAge: sessionExpirationTime})
	router.Use(sessions.Sessions(sessionCookieName, cookieStore))

	router.POST("/backup/check", handlers.BackupCheck)
	router.POST("/backup/asset", handlers.BackupAsset)
	router.POST("/backup/thumb", handlers.BackupAssetThumb)
	router.POST("/bucket/create", handlers.BucketCreate)
	router.POST("/user/create", handlers.UserCreate)
	router.POST("/user/login", handlers.UserLogin)
	router.GET("/asset/list", handlers.AssetList)
	router.GET("/asset/fetch", handlers.AssetFetch)
	router.POST("/asset/delete", handlers.AssetDelete)
	router.GET("/album/list", handlers.AlbumList)
	router.GET("/album/create", handlers.AlbumCreate)
	router.GET("/album/add", handlers.AlbumAddAsset)
	router.Run(GetBindAddress())
}
