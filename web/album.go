package web

import (
	"fmt"
	"net/http"
	"server/db"
	"server/handlers"
	"time"

	"github.com/gin-gonic/gin"
)

type AssetInfo struct {
	ID       uint64
	Type     uint
	MimeType string
}

func getDatesString(min, max int64) string {
	minString := time.Unix(min, 0).Format("2 Jan 2006")
	if max-min <= 86400 {
		return minString
	}
	maxString := time.Unix(max, 0).Format("2 Jan 2006")
	return minString + " - " + maxString
}

func AlbumView(c *gin.Context) {
	token := c.Param("token")
	rows, err := db.Instance.Table("album_shares").Select("album_id, albums.name, users.name").Where("token = ?", token).
		Joins("join albums on album_shares.album_id = albums.id").Joins("join users on album_shares.user_id = users.id").Rows()

	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}

	// Get album info
	var albumId uint64
	var albumName string
	var userName string
	if rows.Next() {
		if err = rows.Scan(&albumId, &albumName, &userName); err != nil {
			fmt.Println(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "something went really wrong"})
			rows.Close()
			return
		}
	}
	rows.Close()

	// Get all assets for the album
	rows, err = db.Instance.Table("album_assets").Select("asset_id, mime_type, assets.created_at").Where("album_id = ?", albumId).Joins("join assets on album_assets.asset_id = assets.id").Order("assets.created_at ASC").Rows()
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AssetInfo{}
	var created, createdMin, createdMax int64
	createdMin = 100000000000
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err = rows.Scan(&assetInfo.ID, &assetInfo.MimeType, &created); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		assetInfo.Type = handlers.GetTypeFrom(assetInfo.MimeType)
		result = append(result, assetInfo)
		if createdMax < created {
			createdMax = created
		}
		if createdMin > created {
			createdMin = created
		}
	}

	c.HTML(http.StatusOK, "album_view.tmpl", gin.H{
		"subtitle": "@" + userName,
		"dates":    getDatesString(createdMin, createdMax),
		"title":    albumName,
		"assets":   result,
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
