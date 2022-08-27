package main

import (
	"server/db"
	"server/locations"
	"server/web"

	// "server/faces"
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
	go locations.StartProcessing()

	// faces.Init("/mnt/data1/models")

	// One off
	// assets := []models.Asset{}
	// res := db.Instance.Table("assets").Where("deleted = 0").Find(&assets)
	// if res.Error != nil {
	// 	fmt.Println(res.Error)
	// 	return
	// }
	// for _, asset := range assets {
	// 	fmt.Printf("Processing Asset: %d\n", asset.ID)
	// 	foundFaces, err := faces.ProcessPhoto(asset.ID, "/mnt/data1/circled-data/"+asset.GetThumbPath())
	// 	if err != nil {
	// 		fmt.Println(err)
	// 		continue
	// 	}
	// 	fmt.Printf("Asset: %d, num faces: %d; saving...\n", asset.ID, len(foundFaces))
	// 	for _, face := range foundFaces {
	// 		db.Instance.Save(&face)
	// 	}
	// }

	router := gin.Default()
	router.SetTrustedProxies([]string{})

	// HTML templates
	router.LoadHTMLGlob("templates/*.tmpl")

	cookieStore := gormsessions.NewStore(db.Instance, true, []byte(sessionStoreKey))
	cookieStore.Options(sessions.Options{MaxAge: sessionExpirationTime})
	router.Use(sessions.Sessions(sessionCookieName, cookieStore))
	// Backup handlers
	router.POST("/backup/check", handlers.BackupCheck)
	router.POST("/backup/asset", handlers.BackupAsset)
	router.POST("/backup/thumb", handlers.BackupAssetThumb)
	// Bucket handlers
	router.POST("/bucket/create", handlers.BucketCreate)
	// User info handlers
	router.POST("/user/create", handlers.UserCreate)
	router.POST("/user/login", handlers.UserLogin)
	router.GET("/user/permissions", handlers.UserGetPermissions)
	// Asset handlers
	router.GET("/asset/list", handlers.AssetList)
	router.GET("/asset/fetch", handlers.AssetFetch)
	router.POST("/asset/delete", handlers.AssetDelete)
	// Album handlers
	router.GET("/album/list", handlers.AlbumList)
	router.POST("/album/create", handlers.AlbumCreate)
	router.GET("/album/add", handlers.AlbumAddAsset)
	router.GET("/album/assets", handlers.AlbumAssets)
	router.GET("/album/share", handlers.AlbumShare)
	// router.GET("/faces/get", handlers.GetFaces)
	// Web interface
	router.GET("/w/album/:token", web.AlbumView)
	router.GET("/w/album/:token/asset", web.AlbumAssetView)

	router.Run(GetBindAddress())
}
