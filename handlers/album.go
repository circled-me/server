package handlers

import (
	"fmt"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/utils"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AlbumInfo struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name"`
	Subtitle    string `json:"subtitle"`
	HeroAssetId uint64 `json:"hero_asset_id"`
}

type AlbumCreateRequest struct {
	Name string `form:"name" binding:"required"`
}

type AlbumAssetRequest struct {
	AlbumID uint64 `form:"album_id" binding:"required"`
	AssetID uint64 `form:"asset_id" binding:"required"`
}

type AlbumViewRequest struct {
	AlbumID uint64 `form:"album_id" binding:"required"`
}

func AlbumList(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.
		Table("albums").
		Select("albums.id, albums.name, albums.hero_asset_id, min(assets.created_at), max(assets.created_at)").
		Joins("join album_assets on album_id = albums.id").
		Joins("join assets on asset_id = assets.id").
		Where("albums.user_id = ?", user.ID).
		Group("albums.id, albums.name, albums.hero_asset_id").
		Order("albums.created_at DESC").
		Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AlbumInfo{}
	var minDate, maxDate int64
	for rows.Next() {
		albumInfo := AlbumInfo{}
		HeroAssetId := &albumInfo.HeroAssetId
		if err = rows.Scan(&albumInfo.ID, &albumInfo.Name, &HeroAssetId, &minDate, &maxDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		albumInfo.Subtitle = utils.GetDatesString(minDate, maxDate)
		result = append(result, albumInfo)
	}
	for i, a := range result {
		if a.HeroAssetId > 0 {
			continue
		}
		// If we don't have default hero image, pick the first one in the album
		// TODO: improve here
		rows, err := db.Instance.Table("album_assets").Select("asset_id").Where("album_id = ?", a.ID).Order("created_at DESC").Limit(1).Rows()
		if err != nil {
			fmt.Println(err)
			continue
		}
		if rows.Next() {
			_ = rows.Scan(&result[i].HeroAssetId)
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, result)
}

func AlbumCreate(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumCreateRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	album := models.Album{
		Name:   r.Name,
		UserID: user.ID,
	}
	result := db.Instance.Create(&album)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, AlbumInfo{
		ID:          album.ID,
		Name:        album.Name,
		HeroAssetId: 0,
	})
}

func AlbumAddAsset(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumAssetRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	albumAsset := models.AlbumAsset{
		AlbumID: r.AlbumID,
		AssetID: r.AssetID,
	}
	result := db.Instance.FirstOrCreate(&albumAsset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func AlbumRemoveAsset(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumAssetRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	albumAsset := models.AlbumAsset{
		AlbumID: r.AlbumID,
		AssetID: r.AssetID,
	}
	result := db.Instance.Delete(&albumAsset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func AlbumAssets(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumViewRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rows, err := db.Instance.
		Table("album_assets").
		Select("asset_id, mime_type").
		Where("album_id = ?", r.AlbumID).
		Joins("join assets on album_assets.asset_id = assets.id").
		Order("assets.created_at ASC").Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AssetInfo{}
	mimeType := ""
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err = rows.Scan(&assetInfo.ID, &mimeType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		assetInfo.Type = GetTypeFrom(mimeType)
		result = append(result, assetInfo)
	}
	c.JSON(http.StatusOK, result)
}

func AlbumShare(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AlbumViewRequest{} // same for now
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	shareInfo := models.NewAlbumShare(user.ID, r.AlbumID)
	shareInfoCond := shareInfo
	shareInfoCond.Token = "" // Token should not be a condition
	result := db.Instance.Where(shareInfoCond).FirstOrCreate(&shareInfo)
	if result.Error != nil {
		fmt.Println(result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"path": "/w/album/" + shareInfo.Token + "/"})
}
