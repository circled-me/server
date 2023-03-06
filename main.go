package main

import (
	"server/db"
	"server/locations"
	"server/video"
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
	go video.StartProcessing()

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
	router.POST("/backup/upload", handlers.BackupUpload)
	router.POST("/backup/meta-data", handlers.BackupMetaData)
	router.POST("/backup/confirm", handlers.BackupConfirm)
	// Bucket handlers
	router.POST("/bucket/create", handlers.BucketCreate)
	// User info handlers
	router.POST("/user/create", handlers.UserCreate)
	router.POST("/user/login", handlers.UserLogin)
	router.GET("/user/permissions", handlers.UserGetPermissions)
	router.GET("/user/list", handlers.UserList)
	// Asset handlers
	router.GET("/asset/list", handlers.AssetList)
	router.GET("/asset/fetch", handlers.AssetFetch)
	router.POST("/asset/delete", handlers.AssetDelete) // TODO: S3
	// Album handlers
	router.GET("/album/list", handlers.AlbumList)
	router.POST("/album/create", handlers.AlbumCreate)
	router.POST("/album/delete", handlers.AlbumDelete)
	router.POST("/album/add", handlers.AlbumAddAsset)
	router.POST("/album/remove", handlers.AlbumRemoveAsset)
	router.GET("/album/assets", handlers.AlbumAssets)
	router.GET("/album/share", handlers.AlbumShare)
	// TODO: there should be a way to list and remove controbutors too
	router.POST("/album/contributor", handlers.AlbumContributor)

	// Upload Request
	router.GET("/upload/share", handlers.UploadShare)
	// Moment handlers
	router.GET("/moment/list", handlers.MomentList)
	router.GET("/moment/assets", handlers.MomentAssets)
	// Group handlers
	// router.GET("/group/list", handlers.GroupList)
	router.POST("/group/create", handlers.GroupCreate)
	router.POST("/group/save", handlers.GroupSave)
	router.POST("/group/delete", handlers.GroupDelete)
	// router.POST("/group/members", handlers.GroupMembers)
	// Face recognition related
	// router.GET("/faces/get", handlers.GetFaces)

	/*
	 *	Web interface
	 */
	// Albums
	router.GET("/w/album/:token/", web.AlbumView)
	router.GET("/w/album/:token/asset", web.AlbumAssetView)
	// File uploads
	router.GET("/w/upload/:token/", web.UploadRequestView)
	router.GET("/w/upload/:token/new-url/", web.UploadRequestNewURL)
	router.POST("/w/upload/:token/confirm/", web.UploadRequestConfirm)
	router.POST("/w/upload/:token/", web.UploadRequestProcess)

	router.Run(GetBindAddress())
}
