package handlers

// import (
// 	"net/http"
// 	"server/auth"
// 	"server/db"
// 	"server/models"

// 	"github.com/gin-gonic/gin"
// )

// type GetFacesRequest struct {
// 	ID uint64 `form:"id" binding:"required"`
// }

// type FaceInfo struct {
// 	ID uint64  `json:"id"`
// 	X1 float32 `json:"x1"`
// 	Y1 float32 `json:"y1"`
// 	X2 float32 `json:"x2"`
// 	Y2 float32 `json:"y2"`
// }

// func GetFaces(c *gin.Context) {
// 	session := auth.LoadSession(c)
// 	userID := session.UserID()
// 	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
// 		return
// 	}
// 	r := GetFacesRequest{}
// 	err := c.ShouldBindQuery(&r)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}
// 	asset := models.Asset{
// 		ID: r.ID,
// 	}
// 	db.Instance.First(&asset)
// 	if asset.ID != r.ID || asset.UserID != userID {
// 		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied 2"})
// 		return
// 	}
// 	// Checks done, fetch faces...
// 	rows, err := db.Instance.Table("faces").Select("id, rect_x1, rect_y1, rect_x2, rect_y2").Where("asset_id = ?", r.ID).Rows()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 1"})
// 		return
// 	}
// 	defer rows.Close()
// 	result := []FaceInfo{}
// 	for rows.Next() {
// 		faceInfo := FaceInfo{}
// 		if err = rows.Scan(&faceInfo.ID, &faceInfo.X1, &faceInfo.Y1, &faceInfo.X2, &faceInfo.Y2); err != nil {
// 			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB error 2"})
// 			return
// 		}
// 		faceInfo.X1 /= float32(asset.ThumbWidth)
// 		faceInfo.Y1 /= float32(asset.ThumbWidth)
// 		faceInfo.X2 /= float32(asset.ThumbWidth)
// 		faceInfo.Y2 /= float32(asset.ThumbWidth)
// 		result = append(result, faceInfo)
// 	}
// 	c.JSON(http.StatusOK, result)
// }
