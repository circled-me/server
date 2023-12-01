package handlers

import (
	"net/http"
	"server/db"
	"server/models"
	"server/utils"
	"strings"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

type MomentInfo struct {
	Places      string `json:"places" form:"places" binding:"required"`
	Name        string `json:"name"`
	Subtitle    string `json:"subtitle"`
	HeroAssetId uint64 `json:"hero_asset_id"`
	Start       int64  `json:"start" form:"start" binding:"required"`
	End         int64  `json:"end" form:"end" binding:"required"`
}

func (m *MomentInfo) merge(a *MomentInfo) {
	m.Start = a.Start

	places := map[string]bool{}
	for _, p := range strings.Split(m.Places, ",") {
		places[p] = true
	}
	for _, p := range strings.Split(a.Places, ",") {
		places[p] = true
	}
	newPlaces := []string{}
	for p := range places {
		newPlaces = append(newPlaces, p)
	}
	m.Places = strings.Join(newPlaces, ",")
	m.Subtitle = utils.GetDatesString(m.Start, m.End)
}

func MomentList(c *gin.Context, user *models.User) {
	// TODO: Minimum number of assets for a location should be configurable (now 6 below)
	rows, err := db.Instance.Raw(`
	select date,
		if(city = '', area, city),
		group_concat(place_id separator ',') places,
		max(hero),
		min(start),
		max(end)
	from   (select place_id,
				from_unixtime(created_at, '%Y-%m-%d') date,
				max(id)								  hero,
				count(*)                              cnt,
				min(created_at)                       start,
				max(created_at)                       end
		from   assets
		where  user_id = ?
				and deleted = 0
				and place_id is not null
		group  by 2, 1
		having cnt > 6
		order  by 2, 1 desc) t
	join places
		on id = place_id
	group  by 1,
		2
	order  by 1 desc,
		2
	`, user.ID).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := []MomentInfo{}
	var date string
	lastMoment := &MomentInfo{}
	for rows.Next() {
		momentInfo := MomentInfo{}
		if err = rows.Scan(&date, &momentInfo.Name, &momentInfo.Places, &momentInfo.HeroAssetId, &momentInfo.Start, &momentInfo.End); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		// Should we merge last moment with this one?
		if lastMoment.Name == momentInfo.Name && lastMoment.Start-momentInfo.End < 2*86400 {
			lastMoment.merge(&momentInfo)
			continue
		}
		momentInfo.Subtitle = utils.GetDatesString(momentInfo.Start, momentInfo.End)
		result = append(result, momentInfo)
		lastMoment = &result[len(result)-1]
	}
	c.JSON(http.StatusOK, result)
}

func MomentAssets(c *gin.Context, user *models.User) {
	r := MomentInfo{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	rows, err := db.Instance.
		Table("assets").
		Select(AssetsSelectClause).
		Joins("LEFT JOIN locations ON locations.gps_lat = truncate(assets.gps_lat, 4) AND locations.gps_long = truncate(assets.gps_long, 4)").
		Where("user_id = ? and place_id in (?) and deleted=0 and created_at>=? and created_at<=?", user.ID, strings.Split(r.Places, ","), r.Start, r.End).
		Order("created_at DESC").Rows()

	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := LoadAssetsFromRows(c, rows)
	if result == nil {
		return
	}
	c.JSON(http.StatusOK, result)
}
