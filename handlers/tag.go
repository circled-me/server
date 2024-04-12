package handlers

import (
	"net/http"
	"server/db"
	"server/models"
	"server/utils"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

const (
	tagTypePlace     = 1
	tagTypePerson    = 2
	tagTypeYear      = 3
	tagTypeMonth     = 4
	tagTypeDay       = 5
	tagTypeSeason    = 6
	tagTypeType      = 7
	tagTypeFavourite = 8
	tagTypeAlbum     = 9
)

type Tag struct {
	Type   int      `json:"t"`
	Value  string   `json:"v"`
	Assets []uint64 `json:"a"`
}
type Tags map[string]Tag

func (t *Tag) toIndex() string {
	return strconv.Itoa(t.Type) + "_" + t.Value
}

func (t *Tags) toArray() []Tag {
	result := []Tag{}
	for _, v := range *t {
		result = append(result, v)
	}
	return result
}

func (t *Tags) add(typ int, val any, assetId uint64) {
	if val == nil {
		return
	}
	tag := Tag{}
	if s, ok := val.(*string); ok && s != nil && *s != "" {
		tag = Tag{typ, *s, []uint64{assetId}}
	} else if st, ok := val.(string); ok && st != "" {
		tag = Tag{typ, st, []uint64{assetId}}
	} else if i, ok := val.(int); ok {
		tag = Tag{typ, strconv.Itoa(i), []uint64{assetId}}
	} else {
		return
	}
	tagIndex := tag.toIndex()
	if _, exists := (*t)[tagIndex]; !exists {
		(*t)[tagIndex] = tag
		return
	}
	tag = (*t)[tagIndex]
	tag.Assets = append(tag.Assets, assetId)
	(*t)[tagIndex] = tag
}

func TagList(c *gin.Context, user *models.User) {
	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.Table("assets").Select("max(updated_at)").Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if isNotModified(c, tx) {
		return
	}
	rows, err := db.Instance.Table("assets").Select("id, mime_type, favourite, created_at, locations.gps_lat, locations.gps_long, area, city, country").
		Where("user_id=? AND deleted=0 AND size>0 AND thumb_size>0", user.ID).
		Joins(LeftJoinForLocations).
		Order("created_at DESC").
		Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	tags := Tags{}
	mimeType := ""
	var assetId uint64
	var createdAt int64
	var gpsLat, gpsLong *float64
	var area, city, country *string
	favourite := false
	for rows.Next() {
		if err = rows.Scan(&assetId, &mimeType, &favourite, &createdAt, &gpsLat, &gpsLong, &area, &city, &country); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		// Add location tags, e.g. "Tokyo", "Matsubara", etc
		tags.add(tagTypePlace, area, assetId)
		tags.add(tagTypePlace, city, assetId)
		tags.add(tagTypePlace, country, assetId)
		// Add time tags, e.g "2023", "April", "22"
		tmpAsset := &models.Asset{
			CreatedAt: createdAt,
			GpsLat:    gpsLat,
			GpsLong:   gpsLong,
		}
		// Time zone for the given GPS coordinates is used
		year, month, day := tmpAsset.GetCreatedTimeInLocation().Date()
		tags.add(tagTypeYear, year, assetId)
		tags.add(tagTypeMonth, month.String(), assetId)
		tags.add(tagTypeDay, day, assetId)
		// Add season
		tags.add(tagTypeSeason, utils.GetSeason(month, gpsLat), assetId)
		// Add type
		if GetTypeFrom(mimeType) == models.AssetTypeVideo {
			tags.add(tagTypeType, "Video", assetId)
		}
		// TODO: add album names?
		// Add favourites
		if favourite {
			tags.add(tagTypeFavourite, "Favourite", assetId)
		}
	}
	result := tags.toArray()
	// Sort tags by popularity (num assets)
	sort.Slice(result, func(i, j int) bool {
		return len(result[i].Assets) > len(result[j].Assets)
	})
	c.JSON(http.StatusOK, result)
}
