// Copyright 2024 Nikolay Dimitrov.
// All rights reserved.
// Use of this source code is governed by a MIT style license that can be found in the LICENSE file.

package main

import (
	"log"
	"server/auth"
	"server/config"
	"server/db"
	"server/processing"
	"server/utils"
	"server/web"
	"strings"
	"time"

	"server/handlers"
	"server/models"
	"server/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	gormsessions "github.com/gin-contrib/sessions/gorm"
	"github.com/gin-gonic/autotls"
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
	processing.Init()
	go processing.StartProcessing()

	// if !config.DEBUG_MODE {
	// 	gin.SetMode(gin.ReleaseMode)
	// }
	router := gin.Default()
	_ = router.SetTrustedProxies([]string{})
	if config.DEBUG_MODE {
		router.Use(utils.ErrorLogMiddleware)
	}
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "POST", "DELETE"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		// AllowOriginFunc: func(origin string) bool {
		// 	return strings.HasSuffix(origin, ".circled.me") || strings.HasSuffix(origin, ".circled.me/")
		// },
		MaxAge: 30 * 24 * time.Hour,
	}))

	// HTML templates
	router.LoadHTMLGlob("templates/*.tmpl")

	cookieStore := gormsessions.NewStore(db.Instance, true, []byte(sessionStoreKey))
	cookieStore.Options(sessions.Options{MaxAge: sessionExpirationTime})
	router.Use(sessions.Sessions(sessionCookieName, cookieStore))
	if !config.DEBUG_MODE {
		router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedPaths([]string{"/asset/fetch"})))
	}
	router.Use((&utils.CacheRouter{CacheTime: utils.CacheNoCache}).Handler()) // No cache by default, individual end-points can override that
	// Custom Auth Router
	authRouter := &auth.Router{Base: router}
	// Backup handlers
	authRouter.POST("/backup/check", handlers.BackupCheck, models.PermissionPhotoUpload, models.PermissionPhotoBackup)
	authRouter.PUT("/backup/upload", handlers.BackupUpload, models.PermissionPhotoUpload)
	authRouter.POST("/backup/meta-data", handlers.BackupMetaData, models.PermissionPhotoUpload)
	authRouter.POST("/backup/confirm", handlers.BackupConfirm, models.PermissionPhotoUpload)
	// Bucket handlers
	authRouter.GET("/bucket/list", handlers.BucketList, models.PermissionAdmin)
	authRouter.POST("/bucket/save", handlers.BucketSave, models.PermissionAdmin)
	// User info handlers
	router.POST("/user/login", handlers.UserLogin)
	authRouter.POST("/user/save", handlers.UserSave, models.PermissionAdmin)
	authRouter.POST("/user/delete", handlers.UserDelete) // PermissionAdmin or own account check (in handler)
	authRouter.POST("/user/reinvite", handlers.UserReInvite, models.PermissionAdmin)
	authRouter.GET("/user/status", handlers.UserGetStatus)
	authRouter.GET("/user/list", handlers.UserList)
	authRouter.POST("/user/logout", handlers.UserLogout)
	// Asset handlers
	authRouter.GET("/asset/list", handlers.AssetList, models.PermissionPhotoUpload)
	authRouter.GET("/asset/tags", handlers.TagList, models.PermissionPhotoUpload)
	authRouter.GET("/asset/fetch", handlers.AssetFetch)                                  // Auth checks are done inside the handler
	authRouter.POST("/asset/delete", handlers.AssetDelete, models.PermissionPhotoUpload) // TODO: S3 Delete done?
	authRouter.POST("/asset/favourite", handlers.AssetFavourite)
	authRouter.POST("/asset/unfavourite", handlers.AssetUnfavourite)
	authRouter.GET("/faces/for-asset", handlers.FacesForAsset, models.PermissionPhotoUpload)
	authRouter.GET("/faces/people", handlers.PeopleList, models.PermissionPhotoUpload)
	authRouter.POST("/faces/create-person", handlers.CreatePerson, models.PermissionPhotoUpload)
	authRouter.POST("/faces/assign", handlers.PersonAssignFace, models.PermissionPhotoUpload)
	// Album handlers
	authRouter.GET("/album/list", handlers.AlbumList)
	authRouter.POST("/album/create", handlers.AlbumCreate, models.PermissionPhotoUpload)
	authRouter.POST("/album/save", handlers.AlbumSave, models.PermissionPhotoUpload) // TODO: Check hero saved?
	authRouter.POST("/album/delete", handlers.AlbumDelete, models.PermissionPhotoUpload)
	authRouter.POST("/album/add", handlers.AlbumAddAssets, models.PermissionPhotoUpload)
	authRouter.POST("/album/remove", handlers.AlbumRemoveAsset, models.PermissionPhotoUpload)
	authRouter.GET("/album/assets", handlers.AlbumAssets)
	authRouter.GET("/album/share", handlers.AlbumShare)
	authRouter.POST("/album/contributor", handlers.AlbumContributorSave, models.PermissionPhotoUpload) // DEPRECATED
	authRouter.GET("/album/contributors", handlers.AlbumContributorsGet, models.PermissionPhotoUpload)
	authRouter.POST("/album/contributors", handlers.AlbumContributorsSave, models.PermissionPhotoUpload)

	// Upload Request
	authRouter.GET("/upload/share", handlers.UploadShare, models.PermissionPhotoUpload)
	// Moment handlers
	authRouter.GET("/moment/list", handlers.MomentList, models.PermissionPhotoUpload)
	authRouter.GET("/moment/assets", handlers.MomentAssets, models.PermissionPhotoUpload)
	// Group handlers
	authRouter.GET("/group/list", handlers.GroupList)
	authRouter.POST("/group/create", handlers.GroupCreate)
	authRouter.POST("/group/save", handlers.GroupSave)
	authRouter.POST("/group/delete", handlers.GroupDelete)

	// WebSocket handler
	authRouter.GET("/ws", handlers.WebSocket)

	// Web albums
	router.GET("/w/album/:token/", web.AlbumView)
	router.GET("/w/album/:token/asset", web.AlbumAssetView)
	// Web file uploads
	router.GET("/w/upload/:token/", web.UploadRequestView)
	router.GET("/w/upload/:token/new-url/", web.UploadRequestNewURL)
	router.POST("/w/upload/:token/confirm/", web.UploadRequestConfirm)
	router.PUT("/w/upload/:token/", web.UploadRequestProcess)
	// Misc
	router.GET("/robots.txt", web.DisallowRobots)

	var err error
	if config.TLS_DOMAINS != "" {
		err = autotls.Run(router, strings.Split(config.TLS_DOMAINS, ",")...)
	} else {
		err = router.Run(config.BIND_ADDRESS)
	}
	log.Fatalf("Server stopped: %v", err)
}
