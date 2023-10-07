package handlers

import (
	"bytes"
	"image"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"server/db"
	"server/models"
	"server/storage"
	"strings"

	"github.com/gin-gonic/gin"
)

type BackupRequest struct {
	RemoteID   string   `json:"id" binding:"required"`
	Name       string   `json:"name" binding:"required"`
	MimeType   string   `json:"mimetype"`
	Lat        *float64 `json:"lat"`
	Long       *float64 `json:"long"`
	Created    int64    `json:"created"`
	Favourite  bool     `json:"favourite"`
	Width      uint16   `json:"width"`
	Height     uint16   `json:"height"`
	Duration   uint32   `json:"duration"`
	TimeOffset *int     `json:"time_offset"`
}

type BackupConfirmation struct {
	ID        uint64 `form:"id" binding:"required"` // Local DB ID
	Size      int64  `form:"size" binding:"required"`
	ThumbSize int64  `form:"thumb_size" binding:""`
}

type BackupUploadRequest struct {
	ID    uint64 `form:"id" binding:"required"` // Local DB ID
	Thumb bool   `form:"thumb"`
}

type BackupCheckRequest struct {
	IDs []string `binding:"required"`
}

type NewMetadataResponse struct {
	ID       uint64 `json:"id"`
	URI      string `json:"uri"`
	Thumb    string `json:"thumb"`
	MimeType string `json:"mime_type"`
}

type BackupAssetResponse struct {
	Error string `json:"error"`
	ID    uint64 `json:"id"`
}

func BackupMetaData(c *gin.Context, user *models.User) {
	var r BackupRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	_ = NewMetadata(c, user, &r)
}

func BackupConfirm(c *gin.Context, user *models.User) {
	var r BackupConfirmation
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{
		ID:        r.ID,
		Size:      r.Size,
		ThumbSize: r.ThumbSize,
	}
	err = db.Instance.Updates(&asset).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
}

func NewMetadata(c *gin.Context, user *models.User, r *BackupRequest) *models.Asset {
	if user.BucketID == nil {
		panic("Bucket is nil")
	}
	if user.Quota > 0 {
		used, _ := user.GetUsage()
		if used > user.Quota {
			c.JSON(http.StatusForbidden, Response{"Quota exceeded"})
			return nil
		}
	}
	asset := models.Asset{
		UserID:     user.ID,
		User:       *user,
		RemoteID:   r.RemoteID,
		Name:       r.Name,
		GroupID:    nil,
		BucketID:   *user.BucketID,
		GpsLat:     r.Lat,
		GpsLong:    r.Long,
		CreatedAt:  r.Created,
		Favourite:  r.Favourite,
		Width:      r.Width,
		Height:     r.Height,
		Duration:   r.Duration,
		TimeOffset: r.TimeOffset,
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

		c.JSON(http.StatusForbidden, Response{"this file type is not allowed"})
		return nil
	}

	result := db.Instance.Create(&asset)
	if result.Error != nil {
		// Try loading the asset by RemoteID, maybe it exists and we should overwrite it
		result = db.Instance.First(&asset, "remote_id = ?", r.RemoteID)
		if result.Error != nil {
			// Now give up...
			c.JSON(http.StatusInternalServerError, DBError1Response)
			return nil
		}
	}
	if db.Instance.Preload("Bucket").First(&asset).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError2Response)
		return nil
	}
	if asset.Favourite {
		fav := models.FavouriteAsset{
			UserID:       user.ID,
			AssetID:      asset.ID,
			AlbumAssetID: nil,
		}
		_ = db.Instance.Create(&fav)
	}
	c.JSON(http.StatusOK, NewMetadataResponse{
		ID:       asset.ID,
		URI:      asset.CreateUploadURI(false, ""),
		Thumb:    asset.CreateUploadURI(true, ""),
		MimeType: asset.MimeType,
	})
	// Save as Paths are updated
	if db.Instance.Save(&asset).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError3Response)
		return nil
	}
	return &asset
}

func BackupUpload(c *gin.Context, user *models.User) {
	BackupLocalAsset(user.ID, c)
}

func BackupLocalAsset(userID uint64, c *gin.Context) {
	var r BackupUploadRequest
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{}
	result := db.Instance.Joins("Bucket").Where("user_id = ? AND assets.id = ?", userID, r.ID).Find(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	thumbContent := bytes.Buffer{}
	reader := io.TeeReader(c.Request.Body, &thumbContent)
	size, err := storage.Save(asset.GetPathOrThumb(r.Thumb), reader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
	if r.Thumb {
		asset.ThumbSize = size
		thumb, _, err := image.Decode(&thumbContent)
		if err != nil {
			c.JSON(http.StatusInternalServerError, Response{err.Error()})
			return
		}
		asset.ThumbWidth = uint16(thumb.Bounds().Dx())
		asset.ThumbHeight = uint16(thumb.Bounds().Dy())
	} else {
		asset.Size = size
	}
	// Re-save asset as we have new .Size, .ThumbWidth, .ThumbHeight
	db.Instance.Updates(&asset)
	c.JSON(http.StatusOK, BackupAssetResponse{"", asset.ID})
}

// BackupCheck returns the ids of all assets that were already uploaded
func BackupCheck(c *gin.Context, user *models.User) {
	var r BackupCheckRequest
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
	rows, err := db.Instance.Table("assets").Select("remote_id").
		Where("user_id = ? AND remote_id IN (?) AND (thumb_size>0 OR (mime_type NOT LIKE 'image/%' AND mime_type NOT LIKE 'video/%'))", user.ID, r.IDs).Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	var remoteID string
	result := []string{}
	for rows.Next() {
		if err = rows.Scan(&remoteID); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		result = append(result, remoteID)
	}
	c.JSON(http.StatusOK, result)
}
