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
	ID uint64 `json:"id"`
}

func AssetList(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhoneBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	rows, err := db.Instance.Table("assets").Select("id").Where("user_id = ?", userID).Order("created_at DESC").Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	result := []AssetInfo{}
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err = rows.Scan(&assetInfo.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, assetInfo)
	}
	c.JSON(http.StatusOK, result)
}

func createThumb(size uint, reader io.Reader, writer io.Writer) error {
	image, _, err := image.Decode(reader)
	if err != nil {
		return err
	}
	newImage := resize.Thumbnail(size, size, image, resize.Lanczos3)
	return jpeg.Encode(writer, newImage, &jpeg.Options{Quality: 90})
}

func AssetFetch(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhoneBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
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
	if asset.ID != r.ID || asset.UserID != userID {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
		return
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	c.Header("content-type", asset.MimeType)
	if r.Thumb == 1 {
		if r.Size == 0 {
			// Default big (1280) size
			_, err = storage.Load(asset.GetThumbPath(), c.Writer)
		} else {
			// Custom size
			var buf bytes.Buffer
			if _, err = storage.Load(asset.GetThumbPath(), &buf); err == nil {
				err = createThumb(r.Size, &buf, c.Writer)
			}
		}
	} else {
		_, err = storage.Load(asset.GetPath(), c.Writer)
	}
	// Handle errors
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
