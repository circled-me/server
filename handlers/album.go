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
	Mode         *uint8 `json:"mode"`
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

type AlbumShareRequest struct {
	AlbumID      uint64 `json:"album_id" form:"album_id" binding:"required"`
	Expires      int64  `json:"expires" form:"expires"` // 0 - Never, or number of seconds from now
	HideOriginal int    `json:"hide_original" form:"hide_original"`
}

type AlbumContributeRequest struct {
	AlbumID uint64 `json:"album_id" binding:"required"`
	UserID  uint64 `json:"user_id" binding:"required"`
	Mode    uint8  `json:"mode"` // 0 - ContributorCanAdd or, 1 - ContributorViewOnly
}

type AlbumContributorsGetRequest struct {
	AlbumID uint64 `form:"album_id" binding:"required"`
}

type AlbumContributors struct {
	AlbumID      uint64           `json:"album_id" binding:"required"`
	Contributors map[uint64]uint8 `json:"contributors" binding:"required"` // id -> mode (0 - ContributorCanAdd or, 1 - ContributorViewOnly)
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
		// TODO: Optimise this
		if user.ID != albumInfo.Owner {
			var mode uint8
			if db.Instance.Raw("select mode from album_contributors where album_id=? and user_id=?", albumInfo.ID, user.ID).Scan(&mode).Error == nil {
				albumInfo.Mode = &mode
			}
		}
		if HeroAssetId != nil {
			albumInfo.HeroAssetId = *HeroAssetId
		}
		albumInfo.Subtitle = utils.GetDatesString(minDate, maxDate)
		result = append(result, albumInfo)
	}
	for i, a := range result {
		if a.HeroAssetId > 0 {
			// TODO: We should save hero id when we add new asset to the album
			continue
		}
		// If we don't have default hero image, pick the first one in the album
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
	result := db.Instance.Raw("select 1 from albums where id=? and (user_id=? OR exists(select 1 from album_contributors where album_contributors.album_id = albums.id and album_contributors.user_id=? and album_contributors.mode=0))", r.AlbumID, user.ID, user.ID).
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
			Select(AssetsSelectClause).
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
			Select(AssetsSelectClause).
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
	result := LoadAssetsFromRows(c, rows)
	if result == nil {
		return
	}
	c.JSON(http.StatusOK, result)
}

func AlbumShare(c *gin.Context, user *models.User) {
	r := AlbumShareRequest{}
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
	shareInfo := models.NewAlbumShare(user.ID, r.AlbumID, r.Expires, r.HideOriginal)
	// Try finding the same share (probably with 0 - 'never expires')
	shareInfoCond := shareInfo
	shareInfoCond.Token = "" // Token should not be a condition
	result = db.Instance.Where(shareInfoCond).Preload("Album").First(&shareInfo)
	if result.Error == nil {
		c.JSON(http.StatusOK, AlbumShareResponse{
			Title: "[ " + shareInfo.Album.Name + " ]",
			Path:  "/w/album/" + shareInfo.Token + "/",
		})
		return
	}
	if db.Instance.Create(&shareInfo).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	if db.Instance.Preload("Album").Find(&shareInfo).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	// TODO: Make below text configurable
	c.JSON(http.StatusOK, AlbumShareResponse{
		Title: "[ " + shareInfo.Album.Name + " ]",
		Path:  "/w/album/" + shareInfo.Token + "/",
	})
}

// AlbumContributorSave is DEPRECATED now
func AlbumContributorSave(c *gin.Context, user *models.User) {
	r := AlbumContributeRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	if r.Mode != models.ContributorCanAdd && r.Mode != models.ContributorViewOnly {
		c.JSON(http.StatusBadRequest, Response{"Invalid share mode"})
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
		Mode:    r.Mode,
	}
	result = db.Instance.Create(&albumContributor)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	// Push notifications in background
	go push.AlbumNewContributor(r.UserID, r.AlbumID, r.Mode, user)

	c.JSON(http.StatusOK, OKResponse)
}

func AlbumContributorsGet(c *gin.Context, user *models.User) {
	r := AlbumContributorsGetRequest{}
	if err := c.ShouldBindWith(&r, binding.Form); err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	album := models.Album{ID: r.AlbumID}
	if db.Instance.First(&album).Error != nil || album.ID != r.AlbumID || album.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, Response{"sorry"})
		return
	}
	rows, err := db.Instance.
		Table("album_contributors").
		Select("user_id, mode").
		Where("album_id = ?", r.AlbumID).
		Order("created_at DESC").
		Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	uid := uint64(0)
	mode := uint8(0)
	result := AlbumContributors{AlbumID: album.ID, Contributors: map[uint64]uint8{}}
	for rows.Next() {
		if err = rows.Scan(&uid, &mode); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		if uid == album.UserID {
			// Skip album owner
			continue
		}
		result.Contributors[uid] = mode
	}
	c.JSON(http.StatusOK, result)
}

func AlbumContributorsSave(c *gin.Context, user *models.User) {
	r := AlbumContributors{}
	if err := c.ShouldBindWith(&r, binding.JSON); err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	album := models.Album{ID: r.AlbumID}
	// Currently only the album owner can edit contributors
	if db.Instance.First(&album).Error != nil || album.ID != r.AlbumID || album.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, Response{"sorry"})
		return
	}
	oldMembers := map[uint64]uint8{}
	// Load current ones
	rows, err := db.Instance.Raw("select user_id, mode from album_contributors where album_id=?", album.ID).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	uid := uint64(0)
	mode := uint8(0)
	for rows.Next() {
		if err = rows.Scan(&uid, &mode); err != nil {
			break
		}
		oldMembers[uid] = mode
	}
	// Delete current contributor assignments
	db.Instance.Exec("delete from album_contributors where album_id=?", album.ID)
	var finalErr error
	for uid, mode := range r.Contributors {
		albumContributor := models.AlbumContributor{
			AlbumID: album.ID,
			UserID:  uid,
			Mode:    mode,
		}
		if err := db.Instance.Create(&albumContributor).Error; err != nil {
			fmt.Printf("Contributor save error: %v", err)
			finalErr = err
		}
		oldMode, existed := oldMembers[uid]
		if !existed || oldMode != mode {
			// Push notifications in background
			go push.AlbumNewContributor(uid, album.ID, mode, user)
		}
	}
	if finalErr != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}
