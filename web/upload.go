package web

import (
	"fmt"
	"net/http"
	"server/db"
	"server/handlers"
	"server/models"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadConfirmation struct {
	ID       uint64 `json:"id" binding:"required"` // Local DB ID
	Size     int64  `json:"size" binding:"required"`
	MimeType string `json:"mime_type" binding:""`
}

type NewAssetResponse struct {
	ID  uint64 `json:"id"`
	URL string `json:"url"`
}

func getUploadRequest(c *gin.Context) (req models.UploadRequest, err error) {
	token := c.Param("token")
	// Valid for 3 hours
	err = db.Instance.
		Where("token = ? and created_at >= "+db.TimestampFunc+"-3*3600", token).
		Preload("User").
		Find(&req).Error
	return
}

func UploadRequestProcess(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went wrong"})
		return
	}
	id, err := strconv.ParseUint(c.Query("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.Response{Error: err.Error()})
		return
	}
	if _, ok := uploadRequestAsset(&req, id); !ok {
		c.JSON(http.StatusForbidden, handlers.NopeResponse)
		return
	}
	handlers.BackupLocalAsset(req.UserID, c)
}

// uploadRequestAsset loads an asset only if it was created through this upload request
func uploadRequestAsset(req *models.UploadRequest, id uint64) (asset models.Asset, ok bool) {
	err := db.Instance.Where("id = ? AND user_id = ?", id, req.UserID).First(&asset).Error
	return asset, err == nil && strings.HasPrefix(asset.RemoteID, req.AssetRemoteIDPrefix())
}

func UploadRequestView(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went wrong"})
		return
	}
	// Some cleanup
	db.Instance.Exec("delete from upload_requests where created_at < " + db.TimestampFunc + "-7200")

	c.HTML(http.StatusOK, "upload_files.tmpl", gin.H{
		"who": "@" + req.User.Name,
	})
}

func UploadRequestNewURL(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went wrong"})
		return
	}
	// TODO: user NewMetadata here too instead of all this
	if req.User.HasNoRemainingQuota() {
		c.JSON(http.StatusForbidden, handlers.Response{Error: "Quota exceeded"})
		return
	}
	asset := models.Asset{
		UserID:    req.UserID,
		BucketID:  *req.User.BucketID,
		RemoteID:  req.AssetRemoteIDPrefix() + strconv.FormatInt(time.Now().UnixNano(), 10),
		Name:      c.Query("name"),
		CreatedAt: time.Now().UnixMilli() / 1000,
	}
	result := db.Instance.Create(&asset)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	if db.Instance.Preload("Bucket").Preload("User").First(&asset).Error != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	response := NewAssetResponse{
		ID:  asset.ID,
		URL: asset.CreateUploadURI(false, req.Token), // TODO: Thumb?
	}
	// Save as Paths are updated
	if db.Instance.Save(&asset).Error != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError2Response)
		return
	}
	c.JSON(http.StatusOK, response)
}

func UploadRequestConfirm(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		c.JSON(http.StatusInternalServerError, handlers.Response{Error: "something went wrong"})
		return
	}
	var r UploadConfirmation
	err = c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, handlers.Response{Error: err.Error()})
		return
	}
	if r.Size < 0 {
		c.JSON(http.StatusBadRequest, handlers.Response{Error: "invalid size"})
		return
	}
	asset, ok := uploadRequestAsset(&req, r.ID)
	if !ok {
		c.JSON(http.StatusForbidden, handlers.NopeResponse)
		return
	}
	size := asset.Size
	if size == 0 {
		// Not measured by us (e.g. direct S3 upload)
		size = r.Size
	}
	mimeType := r.MimeType
	if !handlers.IsAllowedMediaMimeType(mimeType) {
		// Never serve arbitrary client-chosen types (e.g. text/html) from our origin
		mimeType = "application/octet-stream"
	}
	err = db.Instance.Model(&asset).Updates(map[string]interface{}{"size": size, "mime_type": mimeType}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, handlers.DBError1Response)
		return
	}
	c.JSON(http.StatusOK, handlers.OKResponse)
}

func DisallowRobots(c *gin.Context) {
	// c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	c.String(http.StatusOK, "User-agent: facebookexternalhit\nAllow: /w/*\n\n"+"User-agent: *\nDisallow: /\n")
}
