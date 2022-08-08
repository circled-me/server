package handlers

import (
	"fmt"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AlbumInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	HeroAssetId uint64 `json:"hero_asset_id"`
}

type AlbumCreateRequest struct {
	Name string `form:"name" binding:"required"`
}

type AlbumAddRequest struct {
	AlbumID uint64 `form:"album_id" binding:"required"`
	AssetID uint64 `form:"asset_id" binding:"required"`
}

func AlbumList(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.Table("albums").Select("id, name, hero_asset_id").Where("user_id = ?", userID).Order("created_at DESC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AlbumInfo{}
	for rows.Next() {
		albumInfo := AlbumInfo{}
		HeroAssetId := &albumInfo.HeroAssetId
		if err = rows.Scan(&albumInfo.ID, &albumInfo.Name, &HeroAssetId); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, albumInfo)
	}
	for _, a := range result {
		if a.HeroAssetId > 0 {
			continue
		}
		// If we don't have default hero image, pick the first one in the album
		rows, err = db.Instance.Table("album_assets").Select("asset_id").Where("album_id = ?", a.ID).Order("created_at DESC").Limit(1).Rows()
		if err != nil {
			fmt.Println(err)
			continue
		}
		if rows.Next() {
			err = rows.Scan(&a.HeroAssetId)
			fmt.Println(err)
		}
	}
	c.JSON(http.StatusOK, result)
}

func AlbumCreate(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumCreateRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Album{
		Name:   r.Name,
		UserID: userID,
	}
	result := db.Instance.Create(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, AlbumInfo{
		ID:          asset.ID,
		Name:        asset.Name,
		HeroAssetId: 0,
	})
}

func AlbumAddAsset(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumAddRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	albumAsset := models.AlbumAsset{
		AlbumID: r.AlbumID,
		AssetID: r.AssetID,
	}
	result := db.Instance.Find(&albumAsset)
	if result.Error == nil {
		// We already have this record
		c.JSON(http.StatusOK, gin.H{"error": ""})
		return
	}
	result = db.Instance.Create(&albumAsset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}
