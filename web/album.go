package web

import (
	"net/http"
	"server/db"
	"server/handlers"
	"server/utils"

	"github.com/gin-gonic/gin"
)

type AssetInfo struct {
	ID       uint64
	Type     uint
	MimeType string
}

func AlbumView(c *gin.Context) {
	token := c.Param("token")
	rows, err := db.Instance.Table("album_shares").Select("album_id, albums.name, users.name").Where("token = ?", token).
		Joins("join albums on album_shares.album_id = albums.id").Joins("join users on album_shares.user_id = users.id").Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}

	// Get album info
	var albumId uint64
	var albumName string
	var userName string
	if rows.Next() {
		if err = rows.Scan(&albumId, &albumName, &userName); err != nil {
			c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went really wrong"})
			rows.Close()
			return
		}
	}
	rows.Close()

	// Get all assets for the album
	rows, err = db.Instance.Table("album_assets").Select("asset_id, mime_type, assets.created_at").Where("album_id = ?", albumId).Joins("join assets on album_assets.asset_id = assets.id").Order("assets.created_at ASC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	defer rows.Close()
	result := []AssetInfo{}
	var created, createdMin, createdMax int64
	createdMin = 100000000000
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err = rows.Scan(&assetInfo.ID, &assetInfo.MimeType, &created); err != nil {
			c.JSON(http.StatusInternalServerError, handlers.DBError2Response)
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
		"dates":    utils.GetDatesString(createdMin, createdMax),
		"title":    albumName,
		"assets":   result,
	})
}

func AlbumAssetView(c *gin.Context) {
	token := c.Param("token")
	r := handlers.AssetFetchRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.Response{Error: err.Error()})
		return
	}
	// Verify we have permission to view this asset
	rows, err := db.Instance.Table("album_shares").Select("album_assets.album_id").Where("token = ? and album_assets.asset_id = ?", token, r.ID).
		Joins("join album_assets on album_shares.album_id = album_assets.album_id").Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went wrong"})
		return
	}
	defer rows.Close()
	if !rows.Next() {
		c.JSON(http.StatusNotFound, handlers.Response{Error: "something went totally wrong"})
		return
	}
	// Return the asset
	handlers.RealAssetFetch(c, 0)
}
