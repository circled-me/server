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

func getUploadRequest(c *gin.Context) (req models.UploadRequest, err error) {
	token := c.Param("token")
	// TODO: secure it more - need to make sure we have the client ip (proxy protocol?)
	// ip := c.ClientIP()

	// Valid for 1 hour
	err = db.Instance.
		Where("token = ? and created_at >= unix_timestamp()-3600", token).
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
	// fmt.Printf("File: %+v", file)
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
	handlers.UploadAsset(c, &req.User, &backuprequest, fileReader)
}

func UploadRequestView(c *gin.Context) {
	req, err := getUploadRequest(c)
	if err != nil || req.ID == 0 {
		fmt.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "something went wrong"})
		return
	}
	// Some cleanup
	db.Instance.Raw("delete from upload_requests where created_at < unix_timestamp()-7200")

	c.HTML(http.StatusOK, "upload_files.tmpl", gin.H{
		"who": "@" + req.User.Name,
	})
}
