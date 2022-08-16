package web

import (
	"fmt"
	"net/http"
	"server/db"
	"server/handlers"

	"github.com/gin-gonic/gin"
)

func AlbumView(c *gin.Context) {
	token := c.Param("token")
	rows, err := db.Instance.Table("album_shares").Select("album_id, name").Where("token = ?", token).
		Joins("join albums on album_shares.album_id = albums.id").Rows()

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}

	// Get album info
	var albumId uint64
	var albumName string
	if rows.Next() {
		if err = rows.Scan(&albumId, &albumName); err != nil {
			fmt.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "something went really wrong"})
			rows.Close()
			return
		}
	}
	rows.Close()

	// Get all assets for the album
	rows, err = db.Instance.Table("album_assets").Select("asset_id, mime_type").Where("album_id = ?", albumId).Joins("join assets on album_assets.asset_id = assets.id").Order("assets.created_at DESC").Rows()
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []handlers.AssetInfo{}
	mimeType := ""
	for rows.Next() {
		assetInfo := handlers.AssetInfo{}
		if err = rows.Scan(&assetInfo.ID, &mimeType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		assetInfo.Type = handlers.GetTypeFrom(mimeType)
		result = append(result, assetInfo)
	}

	c.HTML(http.StatusOK, "album_view.tmpl", gin.H{
		"title":  albumName,
		"assets": result,
		"token":  token,
	})
}

func AlbumAssetView(c *gin.Context) {
	token := c.Param("token")
	r := handlers.AssetFetchRequest{}
	_ = c.ShouldBindQuery(&r)

	// Verify we have permission to view this asset
	rows, err := db.Instance.Table("album_shares").Select("album_assets.album_id").Where("token = ? and album_assets.asset_id = ?", token, r.ID).
		Joins("join album_assets on album_shares.album_id = album_assets.album_id").Rows()

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	defer rows.Close()
	if !rows.Next() {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went totally wrong"})
		return
	}
	// Return the asset
	handlers.RealAssetFetch(c, 0)
}
