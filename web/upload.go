package web

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"server/db"
	"server/handlers"
	"server/models"
	"server/storage"
	"server/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type UploadConfirmation struct {
	ID       uint64 `form:"id" binding:"required"` // Local DB ID
	Size     int64  `form:"size" binding:"required"`
	MimeType string `form:"mime_type" binding:""`
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
	file, err := c.FormFile("filepond")
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went totally wrong"})
		return
	}
	prefix := req.Token
	if len(prefix) > 10 {
		prefix = prefix[:10]
	}
	backuprequest := handlers.BackupRequest{
		ID:       prefix + "_" + strconv.FormatInt(time.Now().UnixNano(), 10), // TODO: something more unique?
		Name:     file.Filename,
		MimeType: file.Header.Get("content-type"),
	}
	fileReader, err := file.Open()
	if err != nil {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong here"})
		return
	}
	asset := handlers.UploadAsset(c, &req.User, &backuprequest, fileReader)
	if asset == nil || !strings.HasPrefix(asset.MimeType, "image/") {
		return
	}
	// create thumbnail
	bucket := storage.Bucket{ID: asset.BucketID}
	if db.Instance.Find(&bucket).Error != nil {
		log.Printf("missing storage")
		return
	}
	// TODO: move this to photo fix post-processing
	storage := storage.StorageFrom(&bucket)
	var buf, thumb bytes.Buffer
	if _, err = storage.Load(asset.GetPath(), &buf); err != nil {
		log.Printf("missing file or other error: %s", err.Error())
		return
	}
	// TODO: hard-coded?
	imageThumbInfo, err := utils.CreateThumb(1280, &buf, &thumb)
	if err != nil {
		log.Printf("CreateThumb error: %s", err.Error())
		return
	}
	asset.ThumbWidth = imageThumbInfo.NewX
	asset.ThumbHeight = imageThumbInfo.NewY
	asset.Width = imageThumbInfo.OldX
	asset.Height = imageThumbInfo.OldY
	asset.ThumbSize, err = storage.Save(asset.GetThumbPath(), &thumb)
	if err != nil {
		log.Printf("canno save thumb file or other error: %s", err.Error())
		return
	}
	db.Instance.Save(asset)
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
		BucketID: req.User.BucketID,
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
		"url": asset.CreateUploadURI(false),
		// "thumb": asset.CreateUploadURI(true),
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
