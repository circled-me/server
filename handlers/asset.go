package handlers

import (
	"bytes"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
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
	ID uint64 `form:"id" binding:"required"`
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

func AssetList(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type").Where("user_id = ? AND deleted = 0 AND size > 0 AND thumb_size > 0", user.ID).Order("created_at DESC").Rows()
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

func AssetFetch(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	RealAssetFetch(c, user.ID)
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
		// Check if we have access via a Shared Album
		var count int64
		result := db.Instance.Raw("select 1 from album_asset where asset_id=? and exists(select 1 from album_contributors where album_contributors.album_id = album_asset.album_id and album_contributors.user_id=?)", r.ID, checkUser).Scan(&count)
		if result.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
			return
		}
		if count == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
			return
		}
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	if asset.Bucket.IsS3() {
		isThumb := false
		if r.Thumb == 1 {
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

func AssetDelete(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	r := AssetDeleteRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{
		ID: r.ID,
	}
	db.Instance.Joins("Bucket").First(&asset)
	if asset.ID != r.ID || asset.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
		return
	}
	asset.Deleted = true
	err = db.Instance.Save(&asset).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	_ = storage.Delete(asset.GetThumbPath())
	err = storage.Delete(asset.GetPath())
	// Handle errors
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	} else {
		c.JSON(http.StatusOK, gin.H{"error": ""})
	}
}
