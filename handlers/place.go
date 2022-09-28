package handlers

import (
	"net/http"
	"server/auth"
	"server/models"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

type PlaceInfo struct {
	ID      uint64 `json:"id"`
	Country string `json:"country"`
	City    string `json:"city"`
	Area    string `json:"area"`
}

// PlaceList returns a structure of:
// - Favourite places
// - Recent places
// - Popular places
func PlaceList(c *gin.Context) {
	session := auth.LoadSession(c)
	userID := session.UserID()
	if userID == 0 || !session.HasPermission(models.PermissionPhotoBackup) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "access denied"})
		return
	}
	result := make(map[string][]PlaceInfo)
	if favourite := getFavouritePlaces(userID); len(favourite) > 0 {
		result["Favourite"] = favourite
	}
	c.JSON(http.StatusOK, result)
}

func getFavouritePlaces(userID uint64) []PlaceInfo {
	result := make([]PlaceInfo, 0)

	return result
}
