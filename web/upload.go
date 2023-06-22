package web

import (
	"fmt"
	"net/http"
	"server/db"
	"server/handlers"
	"server/models"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadConfirmation struct {
	ID       uint64 `json:"id" binding:"required"` // Local DB ID
	Size     int64  `json:"size" binding:"required"`
	MimeType string `json:"mime_type" binding:""`
}

func getUploadRequest(c *gin.Context) (req models.UploadRequest, err error) {
	token := c.Param("token")
	// TODO: secure it more - need to make sure we have the client ip (proxy protocol?)
	// ip := c.ClientIP()

	// Valid for 3 hours
	err = db.Instance.
		Where("token = ? and created_at >= unix_timestamp()-3*3600", token).
		Preload("User").
		Find(&req).Error
	return
}

func UploadRequestProcess(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	handlers.BackupLocalAsset(req.UserID, c)
}

func UploadRequestView(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	// Some cleanup
	db.Instance.Exec("delete from upload_requests where created_at < unix_timestamp()-7200")

	c.HTML(http.StatusOK, "upload_files.tmpl", gin.H{
		"who": "@" + req.User.Name,
	})
}

func UploadRequestNewURL(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	prefix := req.Token
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	asset := models.Asset{
		UserID:   req.UserID,
		BucketID: *req.User.BucketID,
		RemoteID: prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		Name:     c.Query("name"),
	}
	result := db.Instance.Create(&asset)
	if result.Error != nil {
		c.String(http.StatusInternalServerError, result.Error.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":  asset.ID,
		"url": asset.CreateUploadURI(false, req.Token), // TODO: Thumb?
	})
}

func UploadRequestConfirm(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	var r UploadConfirmation
	err = c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	asset := models.Asset{
		ID:     r.ID,
		UserID: req.UserID,
	}
	result := db.Instance.First(&asset)
	if result.Error != nil {
		c.String(http.StatusInternalServerError, result.Error.Error())
		return
	}
	asset.Size = r.Size
	asset.MimeType = r.MimeType
	db.Instance.Updates(&asset)
}

func DisallowRobots(c *gin.Context) {
	c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
}
