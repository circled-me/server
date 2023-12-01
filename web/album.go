package web

import (
	"net/http"
	"server/db"
	"server/handlers"
	"server/utils"

	"github.com/gin-gonic/gin"
)

func AlbumView(c *gin.Context) {
	token := c.Param("token")
	rows, err := db.Instance.
		Table("album_shares").
		Select("album_id, albums.name, users.name, hide_original, hero_asset_id").
		Where("token = ? and (expires_at is null or expires_at=0 or expires_at>unix_timestamp())", token).
		Joins("join albums on album_shares.album_id = albums.id").
		Joins("join users on album_shares.user_id = users.id").
		Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	// Get album info
	var albumId uint64
	var albumName string
	var userName string
	var hideOriginal int
	var heroAssetID *uint64
	if rows.Next() {
		if err = rows.Scan(&albumId, &albumName, &userName, &hideOriginal, &heroAssetID); err != nil {
			c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went really wrong"})
			rows.Close()
			return
		}
	}
	rows.Close()
	// Get all assets for the album
	rows, err = db.Instance.
		Table("album_assets").
		Select(handlers.AssetsSelectClause).
		Joins("join assets on album_assets.asset_id = assets.id").
		Joins("left join locations on locations.gps_lat = truncate(assets.gps_lat, 4) and locations.gps_long = truncate(assets.gps_long, 4)").
		Where("album_assets.album_id = ? and assets.deleted=0 and assets.size>0 and assets.thumb_size>0", albumId).
		Order("assets.created_at ASC").
		Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	defer rows.Close()
	result := handlers.LoadAssetsFromRows(c, rows)
	if result == nil {
		return
	}
	var createdMin, createdMax uint64
	createdMin = 100000000000
	for i, row := range *result {
		// Hide private information
		(*result)[i].Owner = 0
		(*result)[i].DID = ""
		if hideOriginal > 0 {
			(*result)[i].GpsLat = nil
			(*result)[i].GpsLong = nil
			(*result)[i].Location = nil
		}
		if createdMax < row.Created {
			createdMax = row.Created
		}
		if createdMin > row.Created {
			createdMin = row.Created
		}
	}
	downloadParam := "download"
	if hideOriginal > 0 {
		downloadParam = "thumb"
	}
	json := gin.H{
		"ownerName":     "@" + userName,
		"subtitle":      utils.GetDatesString(int64(createdMin), int64(createdMax)),
		"name":          albumName,
		"assets":        result,
		"downloadParam": downloadParam,
		"heroAssetID":   0,
	}
	if heroAssetID != nil {
		json["heroAssetID"] = *heroAssetID
	}
	if c.Query("format") == "json" {
		c.JSON(http.StatusOK, json)
		return
	}
	c.HTML(http.StatusOK, "album_view.tmpl", json)
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
	hideOriginalCond := ""
	if r.Download == 1 {
		hideOriginalCond = " and album_shares.hide_original = 0"
	}
	rows, err := db.Instance.Table("album_shares").Select("album_assets.album_id").
		Where("token = ? and "+
			"album_assets.asset_id = ? and "+
			"(expires_at is null or expires_at=0 or expires_at>unix_timestamp())"+
			hideOriginalCond, token, r.ID).
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
