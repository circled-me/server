package handlers

import (
	"bytes"
	"image"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"server/auth"
	"server/db"
	"server/models"
	"server/storage"
	"strings"

	"github.com/gin-gonic/gin"
)

type BackupRequest struct {
	ID        string   `form:"id" binding:"required"`
	Name      string   `form:"name" binding:"required"`
	MimeType  string   `form:"mimetype" binding:""`
	Lat       *float64 `form:"lat" binding:""`
	Long      *float64 `form:"long" binding:""`
	Created   int64    `form:"created" binding:""`
	Favourite bool     `form:"favourite" binding:""`
	Width     uint16   `form:"width" binding:""`
	Height    uint16   `form:"height" binding:""`
	Duration  uint32   `form:"duration"`
}

type BackupThumbRequest struct {
	ID string `form:"id" binding:"required"`
}

type BackupCheckRequest struct {
	IDs []string `binding:"required"`
}

func BackupAsset(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_ = UploadAsset(c, &user, &r, c.Request.Body)
}

func UploadAsset(c *gin.Context, user *models.User, r *BackupRequest, reader io.Reader) *models.Asset {
	storage := storage.GetDefaultStorage()
	if storage == nil {
		panic("Storage is nil")
	}
	asset := models.Asset{
		UserID:    user.ID,
		RemoteID:  r.ID,
		Name:      r.Name,
		GroupID:   nil,
		BucketID:  storage.GetBucket().ID,
		GpsLat:    r.Lat,
		GpsLong:   r.Long,
		CreatedAt: r.Created,
		Favourite: r.Favourite,
		Width:     r.Width,
		Height:    r.Height,
		Duration:  r.Duration,
	}
	if r.MimeType != "" {
		asset.MimeType = r.MimeType
	} else {
		// Guess the mime type from the extension
		asset.MimeType = mime.TypeByExtension(filepath.Ext(asset.Name))
	}
	// For now, only allow image and video
	if asset.MimeType != "image/jpeg" &&
		asset.MimeType != "image/png" &&
		asset.MimeType != "image/gif" &&
		asset.MimeType != "image/heic" && // TODO: which?
		asset.MimeType != "image/heif" &&
		!strings.HasPrefix(asset.MimeType, "video/") {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "this file type is not allowed"})
		return nil
	}

	result := db.Instance.Create(&asset)
	if result.Error != nil {
		// Try loading the asset by RemoteID, maybe it exists and we should overwrite it
		result = db.Instance.First(&asset, "remote_id = ?", r.ID)
		if result.Error != nil {
			// Now give up...
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return nil
		}
	}
	var err error
	asset.Size, err = storage.Save(asset.GetPath(), reader)
	if err != nil {
		// We couldn't save the file, delete the DB record too
		db.Instance.Delete(asset)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return nil
	} else if asset.Size <= 0 {
		db.Instance.Delete(asset)
		storage.Delete(asset.GetPath())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty asset"})
		return nil
	}
	// Re-save asset as we have new .Size (TODO: .MimeType)
	db.Instance.Updates(&asset)
	c.JSON(200, gin.H{"error": "", "id": asset.ID})
	return &asset
}

func BackupAssetThumb(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupThumbRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{}
	result := db.Instance.Joins("Bucket").Where("user_id = ? AND remote_id = ?", user.ID, r.ID).Find(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	thumbContent := bytes.Buffer{}
	reader := io.TeeReader(c.Request.Body, &thumbContent)
	asset.ThumbSize, err = storage.Save(asset.GetThumbPath(), reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	thumb, _, err := image.Decode(&thumbContent)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	asset.ThumbWidth = uint16(thumb.Bounds().Dx())
	asset.ThumbHeight = uint16(thumb.Bounds().Dy())
	// Re-save asset as we have new .Size, .ThumbWidth, .ThumbHeight (TODO: .MimeType)
	db.Instance.Updates(&asset)
	c.JSON(200, gin.H{"error": "", "id": asset.ID})
}

// BackupCheck returns the ids of all assets that were already uploaded
func BackupCheck(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupCheckRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	rows, err := db.Instance.Table("assets").Select("remote_id").
		Where("user_id = ? AND remote_id IN (?) AND (thumb_size>0 OR (mime_type NOT LIKE 'image/%' AND mime_type NOT LIKE 'video/%'))", user.ID, r.IDs).Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	var remoteID string
	result := []string{}
	for rows.Next() {
		if err = rows.Scan(&remoteID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, remoteID)
	}
	c.JSON(http.StatusOK, result)
}
