package handlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"server/db"
	"server/models"
	"server/push"
	"server/utils"
	"strings"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-sql-driver/mysql"
)

type AlbumInfo struct {
	ID           uint64 `json:"id"`
	Owner        uint64 `json:"owner"`
	Name         string `json:"name"`
	Subtitle     string `json:"subtitle"`
	HeroAssetId  uint64 `json:"hero_asset_id"`
	Contributors []int  `json:"contributors"`
}

type AlbumSaveRequest struct {
	ID          uint64 `json:"id"`
	Name        string `json:"name" binding:"required"`
	HeroAssetId uint64 `json:"hero_asset_id"`
}

type AlbumAssetsRequest struct {
	AlbumID  uint64   `json:"album_id" binding:"required"`
	AssetIDs []uint64 `json:"asset_ids" binding:"required"`
}

type AlbumIDRequest struct {
	AlbumID uint64 `json:"album_id" form:"album_id" binding:"required"`
}

type AlbumContributeRequest struct {
	AlbumID uint64 `json:"album_id" binding:"required"`
	UserID  uint64 `json:"user_id" binding:"required"`
}

type AlbumShareResponse struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

func getFirstFavouriteAssetID(userID uint64) uint64 {
	fav := models.FavouriteAsset{}
	db.Instance.First(&fav, "user_id = ?", userID)
	return fav.AssetID
}

// AlbumList returns an array of AlbumInfo objects
func AlbumList(c *gin.Context, user *models.User) {
	rows, err := db.Instance.
		Table("albums").
		Select("albums.id, albums.name, albums.user_id, albums.hero_asset_id, ifnull(min(assets.created_at), 0), ifnull(max(assets.created_at), 0)").
		Joins("left join album_contributors on album_contributors.album_id = albums.id").
		Joins("left join album_assets on album_assets.album_id = albums.id").
		Joins("left join assets on asset_id = assets.id").
		Where("albums.user_id = ? OR album_contributors.user_id = ?", user.ID, user.ID).
		Group("albums.id, albums.name, albums.hero_asset_id").
		Order("albums.created_at DESC").
		Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := []AlbumInfo{}
	firstFavourite := getFirstFavouriteAssetID(user.ID)
	if firstFavourite > 0 {
		result = append(result, AlbumInfo{
			ID:          0,
			Owner:       user.ID,
			Name:        "Favourites",
			Subtitle:    "All photos you liked ❤️",
			HeroAssetId: firstFavourite,
		})
	}
	var minDate, maxDate int64
	for rows.Next() {
		albumInfo := AlbumInfo{}
		var HeroAssetId *uint64
		if err = rows.Scan(&albumInfo.ID, &albumInfo.Name, &albumInfo.Owner, &HeroAssetId, &minDate, &maxDate); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		if HeroAssetId != nil {
			albumInfo.HeroAssetId = *HeroAssetId
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

func AlbumCreate(c *gin.Context, user *models.User) {
	r := AlbumSaveRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	r.Name = strings.Trim(r.Name, " \n\t\r")
	if len(r.Name) < 1 {
		c.JSON(http.StatusBadRequest, Response{"empty name"})
		return
	}
	album := models.Album{
		Name:   r.Name,
		UserID: user.ID,
	}
	if r.HeroAssetId > 0 {
		album.HeroAssetID = &r.HeroAssetId
	}
	result := db.Instance.Create(&album)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, AlbumInfo{
		ID:          album.ID,
		Name:        album.Name,
		HeroAssetId: 0,
	})
}

func AlbumSave(c *gin.Context, user *models.User) {
	r := AlbumSaveRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	r.Name = strings.Trim(r.Name, " \n\t\r")
	if len(r.Name) < 1 {
		c.JSON(http.StatusBadRequest, Response{"empty name"})
		return
	}
	if r.ID < 1 {
		c.JSON(http.StatusBadRequest, Response{"no ID"})
		return
	}
	album := models.Album{
		ID: r.ID,
	}
	if db.Instance.Find(&album).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if album.UserID != user.ID {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	album.Name = r.Name
	album.HeroAssetID = &r.HeroAssetId
	result := db.Instance.Save(&album)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError3Response)
		return
	}
	c.JSON(http.StatusOK, AlbumInfo{
		ID:          album.ID,
		Name:        album.Name,
		HeroAssetId: 0,
	})
}

func AlbumDelete(c *gin.Context, user *models.User) {
	r := AlbumIDRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	result := db.Instance.Delete(&models.Album{}, "id=? and user_id=?", r.AlbumID, user.ID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}

func AlbumAddAssets(c *gin.Context, user *models.User) {
	r := AlbumAssetsRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	// Check if this is our album or we are added as a contributor
	count := int64(0)
	result := db.Instance.Raw("select 1 from albums where id=? and (user_id=? OR exists(select 1 from album_contributors where album_contributors.album_id = albums.id and album_contributors.user_id=?))", r.AlbumID, user.ID, user.ID).
		Scan(&count)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if count != 1 {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	failed := []uint64{}
	successful := len(r.AssetIDs)
	for _, id := range r.AssetIDs {
		albumAsset := models.AlbumAsset{
			AlbumID: r.AlbumID,
			AssetID: id,
		}
		result = db.Instance.Create(&albumAsset)
		if result.Error != nil {
			successful--
			if me, ok := result.Error.(*mysql.MySQLError); !ok || me.Number != 1062 {
				failed = append(failed, id)
			}
		}
	}
	// Push notifications in background
	go push.AlbumNewAssets(successful, r.AlbumID, user)

	if len(failed) > 0 {
		c.JSON(http.StatusInternalServerError, MultiResponse{"Some assets cannot be added", failed})
		return
	}
	c.JSON(http.StatusOK, OKMultiResponse)
}

func AlbumRemoveAsset(c *gin.Context, user *models.User) {
	r := AlbumAssetsRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	failed := []uint64{}
	// TODO: Optimise below as it was converted from single asset deletion to multiple
	for _, id := range r.AssetIDs {
		albumAsset := models.AlbumAsset{}
		// Check if this is our album or our asset
		result := db.Instance.Joins("Album").Joins("Asset").Where("album_id=? AND asset_id=?", r.AlbumID, id).Find(&albumAsset)
		if result.Error != nil || result.RowsAffected != 1 {
			failed = append(failed, id)
			continue
		}
		if albumAsset.Album.UserID != user.ID && albumAsset.Asset.UserID != user.ID {
			failed = append(failed, id)
			continue
		}
		// Then we can remove it from the album
		result = db.Instance.Delete(&albumAsset)
		if result.Error != nil {
			failed = append(failed, id)
			continue
		}
	}
	if len(failed) > 0 {
		c.JSON(http.StatusInternalServerError, MultiResponse{"Some assets cannot be removed", failed})
		return
	}
	c.JSON(http.StatusOK, OKMultiResponse)
}

func AlbumAssets(c *gin.Context, user *models.User) {
	r := AlbumIDRequest{}
	_ = c.ShouldBindQuery(&r)

	var err error
	var rows *sql.Rows
	if r.AlbumID == 0 {
		// Favourite album
		rows, err = db.Instance.
			Table("favourite_assets").
			Select(assetsSelectClause).
			Where("favourite_assets.user_id = ?", user.ID).
			Joins("JOIN assets on favourite_assets.asset_id = assets.id").
			Joins("LEFT JOIN locations ON locations.gps_lat = truncate(assets.gps_lat, 4) AND locations.gps_long = truncate(assets.gps_long, 4)").
			Order("assets.created_at DESC").Rows()
	} else {
		// Normal album - check for access (own album or as a contributor)
		access := 0
		db.Instance.Raw("select 1 from albums a left join album_contributors ac on (ac.album_id = a.id) where a.id = ? AND (a.user_id = ? OR ac.user_id = ?)", r.AlbumID, user.ID, user.ID).Scan(&access)
		if access == 0 {
			c.JSON(http.StatusUnauthorized, NopeResponse)
			return
		}
		rows, err = db.Instance.
			Table("album_assets").
			Select(assetsSelectClause).
			Where("album_id = ?", r.AlbumID).
			Joins("JOIN assets on album_assets.asset_id = assets.id").
			Joins("LEFT JOIN locations ON locations.gps_lat = truncate(assets.gps_lat, 4) AND locations.gps_long = truncate(assets.gps_long, 4)").
			Order("assets.created_at DESC").Rows()
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := loadAssetsFromRows(c, rows)
	if result == nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

func AlbumShare(c *gin.Context, user *models.User) {
	r := AlbumIDRequest{} // same for now
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	// Check if this is our album or we are added as a contributor
	var count int64
	result := db.Instance.Raw("select 1 from albums where id=? and (user_id=? OR exists(select 1 from album_contributors where album_contributors.album_id = albums.id and album_contributors.user_id=?))", r.AlbumID, user.ID, user.ID).Scan(&count)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if count != 1 {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	shareInfo := models.NewAlbumShare(user.ID, r.AlbumID)
	shareInfoCond := shareInfo
	shareInfoCond.Token = "" // Token should not be a condition
	result = db.Instance.Where(shareInfoCond).FirstOrCreate(&shareInfo)
	if result.Error != nil {
		fmt.Println(result.Error)
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	db.Instance.Preload("Album").Find(&shareInfo)
	// TODO: Make below configurable
	c.JSON(http.StatusOK, AlbumShareResponse{
		Title: "[ " + shareInfo.Album.Name + " ]",
		Path:  "/w/album/" + shareInfo.Token + "/",
	})
}

func AlbumContributor(c *gin.Context, user *models.User) {
	r := AlbumContributeRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	album := models.Album{
		ID:     r.AlbumID,
		UserID: user.ID,
	}
	result := db.Instance.Find(&album)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if result.RowsAffected != 1 || r.UserID == user.ID {
		c.JSON(http.StatusUnauthorized, Response{"no!"})
		return
	}
	albumContributor := models.AlbumContributor{
		AlbumID: r.AlbumID,
		UserID:  r.UserID,
	}
	result = db.Instance.Create(&albumContributor)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}
