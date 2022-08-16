package handlers

import (
	"bytes"
	"image"
	"io"
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/storage"
	"strconv"
	"strings"

	"image/jpeg"
	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
)

type AssetFetchRequest struct {
	ID    uint64 `form:"id" binding:"required"`
	Thumb uint   `form:"thumb"`
	Size  uint   `form:"size"`
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
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type").Where("user_id = ? AND deleted = 0", userID).Order("created_at DESC").Rows()
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

func createThumb(size uint, reader io.Reader, c *gin.Context) (err error) {
	image, _, err := image.Decode(reader)
	if err != nil {
		return err
	}
	var newBuf bytes.Buffer
	newImage := resize.Thumbnail(size, size, image, resize.Lanczos3)
	if err = jpeg.Encode(&newBuf, newImage, &jpeg.Options{Quality: 90}); err != nil {
		return
	}
	c.Header("content-length", strconv.Itoa(newBuf.Len()))
	_, err = io.Copy(c.Writer, &newBuf)
	return
}

func AssetFetch(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	RealAssetFetch(c, userID)
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
		return
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	c.Header("cache-control", "private, max-age=604800")
	if r.Thumb == 1 {
		c.Header("content-type", "image/jpeg")
		if r.Size == 0 {
			// Default big (1280) size
			_, err = storage.Load(asset.GetThumbPath(), c.Writer)
		} else {
			// Custom size
			var buf bytes.Buffer
			if _, err = storage.Load(asset.GetThumbPath(), &buf); err == nil {
				err = createThumb(r.Size, &buf, c)
			}
		}
	} else {
		c.Header("content-type", asset.MimeType)
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
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
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
	if asset.ID != r.ID || asset.UserID != userID {
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
