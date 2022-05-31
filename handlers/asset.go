package handlers

import (
	"net/http"
	"server/auth"
	"server/db"
	"server/models"
	"server/storage"

	"github.com/gin-gonic/gin"
)

type AssetFetchRequest struct {
	ID uint64 `form:"id" binding:"required"`
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
		c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
		return
	}
	defer rows.Close()
	var id string
	result := []string{}
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
			return
		}
		result = append(result, id)
	}
	c.JSON(http.StatusOK, result)
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
	_, err = storage.Load(asset.GetPath(), c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
	}
}
