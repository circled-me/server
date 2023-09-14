package handlers

import (
	"bytes"
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
	ID   uint64 `json:"id"`
	Type uint   `json:"type"`
}

type AssetDeleteRequest struct {
	IDs []uint64 `json:"ids" binding:"required"`
}

type AssetFavouriteRequest struct {
	ID           uint64 `form:"id" binding:"required"`
	AlbumAssetID uint64 `form:"album_asset_id"`
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

func AssetList(c *gin.Context, user *models.User) {
	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.Table("assets").Select("max(updated_at)").Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if isNotModified(c, tx) {
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type").Where("user_id=? AND deleted=0 AND size>0 AND thumb_size>0", user.ID).Order("created_at DESC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := []AssetInfo{}
	mimeType := ""
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err = rows.Scan(&assetInfo.ID, &mimeType); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		assetInfo.Type = GetTypeFrom(mimeType)
		result = append(result, assetInfo)
	}
	c.JSON(http.StatusOK, result)
}

func AssetFetch(c *gin.Context, user *models.User) {
	RealAssetFetch(c, user.ID)
}

func checkAlbumAccess(c *gin.Context, checkUser, assetID uint64) bool {
	// Check if we have access via a Shared Album
	var count int64
	result := db.Instance.Raw("select 1 from album_assets join album_contributors on (album_contributors.album_id = album_assets.album_id) where album_contributors.user_id=? and asset_id=?", checkUser, assetID).Scan(&count)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return false
	}
	if count == 0 {
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
	if r.Thumb == 1 {
		c.Header("content-type", "image/jpeg")
		if r.Size == 0 {
			// Default big (1280) thumb size
			_, err = storage.Load(asset.GetThumbPath(), c.Writer)
		} else {
			// Custom size
			var buf bytes.Buffer
			if _, err = storage.Load(asset.GetThumbPath(), &buf); err == nil {
				var imageThumbInfo utils.ImageThumbConverted
				imageThumbInfo, err = utils.CreateThumb(r.Size, &buf, c.Writer)
				c.Header("content-length", strconv.FormatInt(imageThumbInfo.ThumbSize, 10))
			}
		}
	} else {
		c.Header("content-type", asset.MimeType)
		if r.Download == 1 {
			c.Header("content-disposition", "attachment; filename=\""+asset.Name+"\"")
		}
		// Handles Byte-ranges too
		storage.Serve(asset.GetPath(), c.Request, c.Writer)
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
		storage := storage.StorageFrom(&asset.Bucket)
		if storage == nil {
			log.Printf("Asset: %d, error: storage is nil", id)
			failed = append(failed, id)
			continue
		}
		// TODO: S3 delete
		if err = storage.Delete(asset.GetThumbPath()); err != nil {
			log.Printf("Asset: %d, thumb delete error: %s", id, err.Error())
		}
		if err = storage.Delete(asset.GetPath()); err != nil {
			log.Printf("Asset: %d, delete error: %s", id, err.Error())
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
	err := c.ShouldBindWith(&r, binding.Form)
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
	err := c.ShouldBindWith(&r, binding.Form)
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
