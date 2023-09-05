package handlers

import (
	"bytes"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"gorm.io/gorm"
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

const (
	tagTypePlace     = 1
	tagTypePerson    = 2
	tagTypeYear      = 3
	tagTypeMonth     = 4
	tagTypeDay       = 5
	tagTypeSeason    = 6
	tagTypeType      = 7
	tagTypeFavourite = 8
	tagTypeAlbum     = 9

	etagHeader = "ETag"
)

type Tag struct {
	Type   int      `json:"t"`
	Value  string   `json:"v"`
	Assets []uint64 `json:"a"`
}
type Tags map[string]Tag

type AssetDeleteRequest struct {
	IDs []uint64 `form:"ids" binding:"required"`
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

func (t *Tag) toIndex() string {
	return strconv.Itoa(t.Type) + "_" + t.Value
}

func (t *Tags) toArray() []Tag {
	result := []Tag{}
	for _, v := range *t {
		result = append(result, v)
	}
	return result
}

func (t *Tags) add(typ int, val any, assetId uint64) {
	if val == nil {
		return
	}
	tag := Tag{}
	if s, ok := val.(*string); ok && s != nil && *s != "" {
		tag = Tag{typ, *s, []uint64{assetId}}
	} else if st, ok := val.(string); ok && st != "" {
		tag = Tag{typ, st, []uint64{assetId}}
	} else if i, ok := val.(int); ok {
		tag = Tag{typ, strconv.Itoa(i), []uint64{assetId}}
	} else {
		return
	}
	tagIndex := tag.toIndex()
	if _, exists := (*t)[tagIndex]; !exists {
		(*t)[tagIndex] = tag
		return
	}
	tag = (*t)[tagIndex]
	tag.Assets = append(tag.Assets, assetId)
	(*t)[tagIndex] = tag
}

func isNotModified(c *gin.Context, tx *gorm.DB) bool {
	// Set the current ETag in all cases
	row := tx.Row()
	updatedAt := uint64(0)
	if err := row.Scan(&updatedAt); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error -1"})
		return false
	}
	c.Header("cache-control", "private, max-age=1")
	c.Header(etagHeader, strconv.FormatUint(updatedAt, 10))

	remoteEtag := c.Request.Header.Get("If-None-Match")
	// Check if remote cache is still valid
	if remoteEtag == "" {
		return false
	}
	// ETag contains last updated asset time
	remoteLastUpdated, _ := strconv.ParseUint(remoteEtag, 10, 64)
	if remoteLastUpdated == updatedAt {
		c.Status(http.StatusNotModified)
		return true
	}
	return false
}

func AssetList(c *gin.Context, user *models.User) {
	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.Table("assets").Select("max(updated_at)").Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if isNotModified(c, tx) {
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type").Where("user_id=? AND deleted=0 AND size>0 AND thumb_size>0", user.ID).Order("created_at DESC").Rows()
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

func TagList(c *gin.Context, user *models.User) {
	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.Table("assets").Select("max(updated_at)").Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if isNotModified(c, tx) {
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type, favourite, created_at, locations.gps_lat, area, city, country").
		Where("user_id=? AND deleted=0 AND size>0 AND thumb_size>0", user.ID).
		Joins("LEFT JOIN locations ON locations.gps_lat = truncate(assets.gps_lat, 4) AND locations.gps_long = truncate(assets.gps_long, 4)").Order("created_at DESC").
		Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	tags := Tags{}
	mimeType := ""
	var assetId, createdAt uint64
	var gpsLat *float32
	var area, city, country *string
	favourite := false
	for rows.Next() {
		if err = rows.Scan(&assetId, &mimeType, &favourite, &createdAt, &gpsLat, &area, &city, &country); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		// Add location tags, e.g. "Tokyo", "Matsubara", etc
		tags.add(tagTypePlace, area, assetId)
		tags.add(tagTypePlace, city, assetId)
		tags.add(tagTypePlace, country, assetId)
		// Add time tags, e.g "2023", "April", "22"
		year, month, day := time.Unix(int64(createdAt), 0).Date()
		tags.add(tagTypeYear, year, assetId)
		tags.add(tagTypeMonth, month.String(), assetId)
		tags.add(tagTypeDay, day, assetId)
		// Add season
		tags.add(tagTypeSeason, utils.GetSeason(month, gpsLat), assetId)
		// Add type
		if GetTypeFrom(mimeType) == models.AssetTypeVideo {
			tags.add(tagTypeType, "Video", assetId)
		}
		// TODO: add album names?
		// Add favourites
		if favourite {
			tags.add(tagTypeFavourite, "Favourite", assetId)
		}
	}
	result := tags.toArray()
	sort.Slice(result, func(i, j int) bool {
		return len(result[i].Assets) > len(result[j].Assets)
	})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return false
	}
	if count == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
		return false
	}
	return true
}

func RealAssetFetch(c *gin.Context, checkUser uint64) {
	r := AssetFetchRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func AssetDelete(c *gin.Context, user *models.User) {
	r := AssetDeleteRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Some assets cannot be deleted", "failed": failed})
	} else {
		c.JSON(http.StatusOK, gin.H{"error": "", "failed": failed})
	}
}

func AssetFavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{ID: r.ID}
	db.Instance.First(&asset)
	if asset.ID != r.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
		return
	}
	if r.AlbumAssetID == 0 || asset.UserID == user.ID {
		r.AlbumAssetID = 0
		// This must be our own asset
		if asset.ID != r.ID || asset.UserID != user.ID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
			return
		}
	} else {
		// We should have access to this album
		albumAsset := models.AlbumAsset{ID: r.AlbumAssetID}
		db.Instance.First(&albumAsset)
		if albumAsset.ID != r.AlbumAssetID || albumAsset.AssetID != r.ID {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 3"})
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error 5"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}

func AssetUnfavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindWith(&r, binding.Form)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	fav := models.FavouriteAsset{}
	err = db.Instance.First(&fav, "user_id=? AND asset_id=?", user.ID, r.ID).Error
	if err != nil || fav.UserID != user.ID || fav.AssetID != r.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "some error 2"})
		return
	}
	if db.Instance.Delete(&fav).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error 3"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"error": ""})
}
