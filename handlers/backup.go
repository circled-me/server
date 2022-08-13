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

	"github.com/gin-gonic/gin"
)

type BackupRequest struct {
	ID        string   `form:"id" binding:"required"`
	Name      string   `form:"name" binding:"required"`
	MimeType  string   `form:"mimetype" binding:""`
	Lat       *float64 `form:"lat" binding:""`
	Long      *float64 `form:"long" binding:""`
	Created   uint64   `form:"created" binding:""`
	Favourite bool     `form:"favourite" binding:""`
	Width     uint16   `form:"width" binding:""`
	Height    uint16   `form:"height" binding:""`
	Duration  uint16   `form:"duration"`
}

type BackupThumbRequest struct {
	ID string `form:"id" binding:"required"`
}

type BackupCheckRequest struct {
	IDs []string `binding:"required"`
}

func BackupAsset(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	storage := storage.GetDefaultStorage()
	if storage == nil {
		panic("Storage is nil")
	}
	asset := models.Asset{
		UserID:    userID,
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
	result := db.Instance.Create(&asset)
	if result.Error != nil {
		// Try loading the asset by RemoteID, maybe it exists and we should overwrite it
		result = db.Instance.First(&asset, "remote_id = ?", r.ID)
		if result.Error != nil {
			// Now give up...
			c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
			return
		}
	}
	asset.Size, err = storage.Save(asset.GetPath(), c.Request.Body)
	if err != nil {
		// We couldn't save the file, delete the DB record too
		db.Instance.Delete(asset)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	} else if asset.Size <= 0 {
		db.Instance.Delete(asset)
		storage.Delete(asset.GetPath())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "empty asset"})
		return
	}
	// Re-save asset as we have new .Size (TODO: .MimeType)
	db.Instance.Updates(&asset)
	c.JSON(200, gin.H{"error": "", "id": asset.ID})
}

func BackupAssetThumb(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupThumbRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	storage := storage.GetDefaultStorage()
	if storage == nil {
		panic("Storage is nil")
	}
	asset := models.Asset{}
	result := db.Instance.Where("user_id = ? AND remote_id = ?", userID, r.ID).Find(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
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
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
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
		Where("user_id = ? AND remote_id IN (?) AND (thumb_size>0 OR (mime_type NOT LIKE 'image/%' AND mime_type NOT LIKE 'video/%'))", userID, r.IDs).Rows()

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
