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
	ID        string   `form:"id" binding:"required"` // Remote asset ID (string)
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

type BackupConfirmation struct {
	ID        uint64 `form:"id" binding:"required"` // Local DB ID
	Size      int64  `form:"size" binding:"required"`
	ThumbSize int64  `form:"thumb_size" binding:""`
}

type BackupUploadRequest struct {
	ID uint64 `form:"id" binding:"required"` // Local DB ID
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

func BackupMetaData(c *gin.Context) {
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
	_ = NewMetadata(c, &user, &r)
}

func BackupConfirm(c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupConfirmation
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{
		ID:        r.ID,
		Size:      r.Size,
		ThumbSize: r.ThumbSize,
	}
	err = db.Instance.Updates(&asset).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
}

func NewMetadata(c *gin.Context, user *models.User, r *BackupRequest) *models.Asset {
	if user.BucketID == 0 {
		panic("Bucket is nil")
	}
	asset := models.Asset{
		UserID:    user.ID,
		RemoteID:  r.ID,
		Name:      r.Name,
		GroupID:   nil,
		BucketID:  user.BucketID,
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
		asset.MimeType != "image/heic" && // TODO: which one to remain?
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
	if db.Instance.Preload("Bucket").First(&asset).Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error 3"})
		return nil
	}
	c.JSON(http.StatusOK, gin.H{
		"id":        asset.ID,
		"uri":       asset.CreateUploadURI(false),
		"thumb":     asset.CreateUploadURI(true),
		"mime_type": asset.MimeType,
	})
	return &asset
}

// TODO: DEPRECATE
func UploadAsset(c *gin.Context, user *models.User, r *BackupRequest, reader io.Reader) *models.Asset {
	db.Instance.Preload("Bucket").First(&user)
	storage := storage.StorageFrom(&user.Bucket)
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
	c.JSON(http.StatusOK, gin.H{"error": "", "id": asset.ID})
	return &asset
}

func BackupUpload(c *gin.Context) {
	backupLocalAsset(false, c)
}

func BackupAssetThumb(c *gin.Context) {
	backupLocalAsset(true, c)
}

func backupLocalAsset(isThumb bool, c *gin.Context) {
	session := auth.LoadSession(c)
	user := session.User()
	if user.ID == 0 || !user.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	var r BackupUploadRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{}
	result := db.Instance.Joins("Bucket").Where("user_id = ? AND id = ?", user.ID, r.ID).Find(&asset)
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
	size, err := storage.Save(asset.GetPathOrThumb(isThumb), reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if isThumb {
		asset.ThumbSize = size
		thumb, _, err := image.Decode(&thumbContent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		asset.ThumbWidth = uint16(thumb.Bounds().Dx())
		asset.ThumbHeight = uint16(thumb.Bounds().Dy())
	} else {
		asset.Size = size
	}
	// Re-save asset as we have new .Size, .ThumbWidth, .ThumbHeight (TODO: .MimeType)
	db.Instance.Updates(&asset)
	c.JSON(http.StatusOK, gin.H{"error": "", "id": asset.ID})
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
