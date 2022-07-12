package handlers

import (
	"net/http"
	"server/auth"
	"server/db"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

type AlbumInfo struct {
	ID   uint64 `json:"id"`
	Name string `json:"name"`
}

type AlbumCreateRequest struct {
	Name string `form:"name"`
}

type AlbumAddRequest struct {
	AlbumID uint64 `form:"album_id"`
	AssetID uint64 `form:"asset_id"`
}

func AlbumList(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.Table("albums").Select("id, name").Where("user_id = ?", userID).Order("created_at DESC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AlbumInfo{}
	for rows.Next() {
		albumInfo := AlbumInfo{}
		if err = rows.Scan(&albumInfo.ID, &albumInfo.Name); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, albumInfo)
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
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Album{
		Name: r.Name,
	}
	result := db.Instance.Create(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": asset.ID, "name": asset.Name})
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
	result := db.Instance.Create(&albumAsset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}
