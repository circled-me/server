package handlers

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AssetFetchRequest struct {
	ID       uint64 `form:"id" binding:"required"`
	Thumb    uint   `form:"thumb"`
	Download uint   `form:"download"`
	Size     uint   `form:"size"`
}

type AssetInfo struct {
	ID        uint64   `json:"id"`
	Type      uint     `json:"type"`
	Owner     uint64   `json:"owner"`
	Name      string   `json:"name"`
	Location  *string  `json:"location"`
	DID       string   `json:"did"` // DeviceID
	Created   uint64   `json:"created"`
	GpsLat    *float64 `json:"gps_lat"`
	GpsLong   *float64 `json:"gps_long"`
	Size      uint64   `json:"size"`
	MimeType  string   `json:"mime_type"`
	Favourite bool     `json:"favourite"`
}

const (
	// created_at field is adjusted with time_offset so the time can be shown "as UTC"
	AssetsSelectClause   = "assets.id, assets.name, assets.user_id, assets.created_at+ifnull(time_offset,0), assets.remote_id, assets.mime_type, assets.gps_lat, assets.gps_long, locations.display, assets.size, assets.mime_type, favourite_assets.asset_id is not null as f"
	LeftJoinForLocations = "left join locations ON locations.gps_lat = round(assets.gps_lat*10000-0.5)/10000.0 AND locations.gps_long = round(assets.gps_long*10000-0.5)/10000.0"
)

type AssetDeleteRequest struct {
	IDs []uint64 `json:"ids" binding:"required"`
}

type AssetFavouriteRequest struct {
	ID           uint64 `json:"id" binding:"required"`
	AlbumAssetID uint64 `json:"album_asset_id"`
}

// TODO: Move to before save in Asset
func GetTypeFrom(mimeType string) uint {
	if strings.HasPrefix(mimeType, "image/") {
		return models.AssetTypeImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return models.AssetTypeVideo
	}
	return models.AssetTypeOther
}

func LoadAssetsFromRows(c *gin.Context, rows *sql.Rows) *[]AssetInfo {
	result := []AssetInfo{}
	mimeType := ""
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err := rows.Scan(&assetInfo.ID, &assetInfo.Name, &assetInfo.Owner, &assetInfo.Created, &assetInfo.DID, &mimeType,
			&assetInfo.GpsLat, &assetInfo.GpsLong, &assetInfo.Location, &assetInfo.Size, &assetInfo.MimeType, &assetInfo.Favourite); err != nil {

			log.Printf("DB error: %v", err)
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return nil
		}
		assetInfo.Type = GetTypeFrom(mimeType)
		result = append(result, assetInfo)
	}
	return &result
}

func AssetList(c *gin.Context, user *models.User) {
	fr := AssetsForFaceRequest{}
	_ = c.ShouldBindQuery(&fr)

	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.
		Table("assets").
		Select("max(updated_at)").
		Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if fr.FaceID == 0 && c.Query("reload") != "1" && isNotModified(c, tx) {
		return
	}
	// TODO: For big sets maybe dynamically load asset info individually?
	tmp := db.Instance.
		Table("assets").
		Select(AssetsSelectClause).
		Joins("left join favourite_assets on favourite_assets.asset_id = assets.id").
		Joins(LeftJoinForLocations)
	if fr.FaceID > 0 {
		// Find assets with faces similar to the given face or with the same person already assigned
		tmp = tmp.Joins("join (select distinct t2.asset_id from faces t1 join faces t2 where t1.id=? and (t1.person_id = t2.person_id OR "+models.FacesVectorDistance+" <= ?)) f on f.asset_id = assets.id", fr.FaceID, fr.Threshold)
	}
	rows, err := tmp.
		Where("assets.user_id=? and assets.deleted=0 and assets.size>0 and assets.thumb_size>0", user.ID).Order("assets.created_at DESC").Rows()
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

func AssetFetch(c *gin.Context, user *models.User) {
	RealAssetFetch(c, user.ID)
}

func checkAlbumAccess(c *gin.Context, checkUser, assetID uint64) bool {
	// Check if we have access via any shared album or if any of those albums is ours
	var sum int64
	result := db.Instance.Raw("select sum(ifnull(album_contributors.user_id, ifnull(albums.user_id, 0))) "+
		"from album_assets "+
		"left join album_contributors on (album_contributors.album_id = album_assets.album_id and album_contributors.user_id = ?) "+
		"left join albums on (albums.id = album_assets.album_id and albums.user_id = ?) "+
		"where asset_id=?", checkUser, checkUser, assetID).Scan(&sum)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return false
	}
	if sum == 0 {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return false
	}
	return true
}

func RealAssetFetch(c *gin.Context, checkUser uint64) {
	r := AssetFetchRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{
		ID: r.ID,
	}
	db.Instance.Joins("Bucket").First(&asset)
	if checkUser > 0 && asset.UserID != checkUser {
		if !checkAlbumAccess(c, checkUser, r.ID) {
			return
		}
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	if asset.Bucket.IsS3() {
		isThumb := false
		if r.Thumb == 1 && asset.ThumbSize > 0 {
			isThumb = true
		}
		// Redirect to the S3 location
		url, expires := asset.GetS3DownloadURL(isThumb)
		maxAge := expires - time.Now().Unix()
		c.Header("cache-control", "private, max-age="+strconv.FormatInt(maxAge, 10))
		c.Redirect(302, url)
		return
	}
	c.Header("cache-control", "private, max-age=604800")
	if r.Thumb == 1 && asset.ThumbSize > 0 {
		c.Header("content-type", "image/jpeg")
		if r.Size == 0 {
			// Default big (1280) thumb size
			_, err = storage.Load(asset.ThumbPath, c.Writer)
		} else {
			// Custom size
			var buf bytes.Buffer
			if _, err = storage.Load(asset.ThumbPath, &buf); err == nil {
				var imageThumbInfo utils.ImageThumbConverted
				imageThumbInfo, err = utils.CreateThumb(r.Size, &buf, c.Writer)
				c.Header("content-length", strconv.FormatInt(imageThumbInfo.ThumbSize, 10))
			}
		}
	} else {
		// Original
		c.Header("content-type", asset.MimeType)
		if r.Download == 1 {
			c.Header("content-disposition", "attachment; filename=\""+asset.Name+"\"")
		}
		// Handles Byte-ranges too
		storage.Serve(asset.Path, c.Request, c.Writer)
		return
	}
	// Handle errors
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
	}
}

func AssetDelete(c *gin.Context, user *models.User) {
	r := AssetDeleteRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	failed := []uint64{}
	for _, id := range r.IDs {
		asset := models.Asset{
			ID: id,
		}
		db.Instance.Joins("Bucket").First(&asset)
		if asset.ID != id || asset.UserID != user.ID {
			failed = append(failed, id)
			log.Printf("Asset: %d, auth error", id)
			continue
		}
		asset.Deleted = true
		err = db.Instance.Save(&asset).Error
		if err != nil {
			failed = append(failed, id)
			log.Printf("Asset: %d, save error %s", id, err)
			continue
		}
		// TODO: Delete record better (and rely on cascaded deletes) and reinsert with same RemoteID (to stop re-uploading)?
		db.Instance.Exec("delete from album_assets where asset_id=?", id)
		db.Instance.Exec("delete from favourite_assets where asset_id=?", id)
		db.Instance.Exec("delete from faces where asset_id=?", id)
		storage := storage.StorageFrom(&asset.Bucket)
		if storage == nil {
			log.Printf("Asset: %d, error: storage is nil", id)
			failed = append(failed, id)
			continue
		}
		// Finally delete
		if err = storage.Delete(asset.ThumbPath); err != nil {
			log.Printf("Asset: %d, thumb delete error: %s", id, err.Error())
		}
		if err = storage.Delete(asset.Path); err != nil {
			log.Printf("Asset: %d, delete error: %s", id, err.Error())
		}
		// Remote (S3) as well
		if err = storage.DeleteRemoteFile(asset.ThumbPath); err != nil {
			log.Printf("Remote Asset: %d, thumb delete error: %s", id, err.Error())
		}
		if err = storage.DeleteRemoteFile(asset.Path); err != nil {
			log.Printf("Remote Asset: %d, delete error: %s", id, err.Error())
		}
	}
	// Handle errors
	if len(failed) > 0 {
		c.JSON(http.StatusInternalServerError, MultiResponse{"Some assets cannot be deleted", failed})
		return
	}
	c.JSON(http.StatusOK, OKMultiResponse)
}

func AssetFavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{ID: r.ID}
	db.Instance.First(&asset)
	if asset.ID != r.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	if r.AlbumAssetID == 0 || asset.UserID == user.ID {
		r.AlbumAssetID = 0
		// This must be our own asset
		if asset.ID != r.ID || asset.UserID != user.ID {
			c.JSON(http.StatusUnauthorized, Nope2Response)
			return
		}
	} else {
		// We should have access to this album
		albumAsset := models.AlbumAsset{ID: r.AlbumAssetID}
		db.Instance.First(&albumAsset)
		if albumAsset.ID != r.AlbumAssetID || albumAsset.AssetID != r.ID {
			c.JSON(http.StatusUnauthorized, Nope3Response)
			return
		}
		if !checkAlbumAccess(c, user.ID, r.ID) {
			return
		}
	}
	// All checks done! Phew...
	fav := models.FavouriteAsset{
		UserID:       user.ID,
		AssetID:      r.ID,
		AlbumAssetID: nil,
	}
	if r.AlbumAssetID > 0 {
		fav.AlbumAssetID = &r.AlbumAssetID
	}
	if db.Instance.Create(&fav).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}

func AssetUnfavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	fav := models.FavouriteAsset{}
	err = db.Instance.First(&fav, "user_id=? AND asset_id=?", user.ID, r.ID).Error
	if err != nil || fav.UserID != user.ID || fav.AssetID != r.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	if db.Instance.Delete(&fav).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError3Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}
