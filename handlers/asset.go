package handlers

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strconv"
	"strings"
	"time"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type AssetFetchRequest struct {
	ID       uint64 `form:"id" binding:"required"`
	Thumb    uint   `form:"thumb"`
	Download uint   `form:"download"`
	Size     uint   `form:"size"`
}

type AssetInfo struct {
	ID        uint64   `json:"id"`
	Type      uint     `json:"type"`
	Owner     uint64   `json:"owner"`
	Name      string   `json:"name"`
	Location  *string  `json:"location"`
	DID       string   `json:"did"` // DeviceID
	Created   uint64   `json:"created"`
	GpsLat    *float64 `json:"gps_lat"`
	GpsLong   *float64 `json:"gps_long"`
	Size      uint64   `json:"size"`
	MimeType  string   `json:"mime_type"`
	Favourite bool     `json:"favourite"`
}

const (
	// created_at field is adjusted with time_offset so the time can be shown "as UTC"
	AssetsSelectClause   = "assets.id, assets.name, assets.user_id, assets.created_at+ifnull(time_offset,0), assets.remote_id, assets.mime_type, assets.gps_lat, assets.gps_long, locations.display, assets.size, assets.mime_type, favourite_assets.asset_id is not null as f"
	LeftJoinForLocations = "left join locations ON locations.gps_lat = round(assets.gps_lat*10000-0.5)/10000.0 AND locations.gps_long = round(assets.gps_long*10000-0.5)/10000.0"
	vectorDotProduct     = "(t2.v0-t1.v0)*(t2.v0-t1.v0) + (t2.v1-t1.v1)*(t2.v1-t1.v1) + (t2.v2-t1.v2)*(t2.v2-t1.v2) + (t2.v3-t1.v3)*(t2.v3-t1.v3) + (t2.v4-t1.v4)*(t2.v4-t1.v4) + (t2.v5-t1.v5)*(t2.v5-t1.v5) + (t2.v6-t1.v6)*(t2.v6-t1.v6) + (t2.v7-t1.v7)*(t2.v7-t1.v7) + (t2.v8-t1.v8)*(t2.v8-t1.v8) + (t2.v9-t1.v9)*(t2.v9-t1.v9) + (t2.v10-t1.v10)*(t2.v10-t1.v10) + (t2.v11-t1.v11)*(t2.v11-t1.v11) + (t2.v12-t1.v12)*(t2.v12-t1.v12) + (t2.v13-t1.v13)*(t2.v13-t1.v13) + (t2.v14-t1.v14)*(t2.v14-t1.v14) + (t2.v15-t1.v15)*(t2.v15-t1.v15) + (t2.v16-t1.v16)*(t2.v16-t1.v16) + (t2.v17-t1.v17)*(t2.v17-t1.v17) + (t2.v18-t1.v18)*(t2.v18-t1.v18) + (t2.v19-t1.v19)*(t2.v19-t1.v19) + (t2.v20-t1.v20)*(t2.v20-t1.v20) + (t2.v21-t1.v21)*(t2.v21-t1.v21) + (t2.v22-t1.v22)*(t2.v22-t1.v22) + (t2.v23-t1.v23)*(t2.v23-t1.v23) + (t2.v24-t1.v24)*(t2.v24-t1.v24) + (t2.v25-t1.v25)*(t2.v25-t1.v25) + (t2.v26-t1.v26)*(t2.v26-t1.v26) + (t2.v27-t1.v27)*(t2.v27-t1.v27) + (t2.v28-t1.v28)*(t2.v28-t1.v28) + (t2.v29-t1.v29)*(t2.v29-t1.v29) + (t2.v30-t1.v30)*(t2.v30-t1.v30) + (t2.v31-t1.v31)*(t2.v31-t1.v31) + (t2.v32-t1.v32)*(t2.v32-t1.v32) + (t2.v33-t1.v33)*(t2.v33-t1.v33) + (t2.v34-t1.v34)*(t2.v34-t1.v34) + (t2.v35-t1.v35)*(t2.v35-t1.v35) + (t2.v36-t1.v36)*(t2.v36-t1.v36) + (t2.v37-t1.v37)*(t2.v37-t1.v37) + (t2.v38-t1.v38)*(t2.v38-t1.v38) + (t2.v39-t1.v39)*(t2.v39-t1.v39) + (t2.v40-t1.v40)*(t2.v40-t1.v40) + (t2.v41-t1.v41)*(t2.v41-t1.v41) + (t2.v42-t1.v42)*(t2.v42-t1.v42) + (t2.v43-t1.v43)*(t2.v43-t1.v43) + (t2.v44-t1.v44)*(t2.v44-t1.v44) + (t2.v45-t1.v45)*(t2.v45-t1.v45) + (t2.v46-t1.v46)*(t2.v46-t1.v46) + (t2.v47-t1.v47)*(t2.v47-t1.v47) + (t2.v48-t1.v48)*(t2.v48-t1.v48) + (t2.v49-t1.v49)*(t2.v49-t1.v49) + (t2.v50-t1.v50)*(t2.v50-t1.v50) + (t2.v51-t1.v51)*(t2.v51-t1.v51) + (t2.v52-t1.v52)*(t2.v52-t1.v52) + (t2.v53-t1.v53)*(t2.v53-t1.v53) + (t2.v54-t1.v54)*(t2.v54-t1.v54) + (t2.v55-t1.v55)*(t2.v55-t1.v55) + (t2.v56-t1.v56)*(t2.v56-t1.v56) + (t2.v57-t1.v57)*(t2.v57-t1.v57) + (t2.v58-t1.v58)*(t2.v58-t1.v58) + (t2.v59-t1.v59)*(t2.v59-t1.v59) + (t2.v60-t1.v60)*(t2.v60-t1.v60) + (t2.v61-t1.v61)*(t2.v61-t1.v61) + (t2.v62-t1.v62)*(t2.v62-t1.v62) + (t2.v63-t1.v63)*(t2.v63-t1.v63) + (t2.v64-t1.v64)*(t2.v64-t1.v64) + (t2.v65-t1.v65)*(t2.v65-t1.v65) + (t2.v66-t1.v66)*(t2.v66-t1.v66) + (t2.v67-t1.v67)*(t2.v67-t1.v67) + (t2.v68-t1.v68)*(t2.v68-t1.v68) + (t2.v69-t1.v69)*(t2.v69-t1.v69) + (t2.v70-t1.v70)*(t2.v70-t1.v70) + (t2.v71-t1.v71)*(t2.v71-t1.v71) + (t2.v72-t1.v72)*(t2.v72-t1.v72) + (t2.v73-t1.v73)*(t2.v73-t1.v73) + (t2.v74-t1.v74)*(t2.v74-t1.v74) + (t2.v75-t1.v75)*(t2.v75-t1.v75) + (t2.v76-t1.v76)*(t2.v76-t1.v76) + (t2.v77-t1.v77)*(t2.v77-t1.v77) + (t2.v78-t1.v78)*(t2.v78-t1.v78) + (t2.v79-t1.v79)*(t2.v79-t1.v79) + (t2.v80-t1.v80)*(t2.v80-t1.v80) + (t2.v81-t1.v81)*(t2.v81-t1.v81) + (t2.v82-t1.v82)*(t2.v82-t1.v82) + (t2.v83-t1.v83)*(t2.v83-t1.v83) + (t2.v84-t1.v84)*(t2.v84-t1.v84) + (t2.v85-t1.v85)*(t2.v85-t1.v85) + (t2.v86-t1.v86)*(t2.v86-t1.v86) + (t2.v87-t1.v87)*(t2.v87-t1.v87) + (t2.v88-t1.v88)*(t2.v88-t1.v88) + (t2.v89-t1.v89)*(t2.v89-t1.v89) + (t2.v90-t1.v90)*(t2.v90-t1.v90) + (t2.v91-t1.v91)*(t2.v91-t1.v91) + (t2.v92-t1.v92)*(t2.v92-t1.v92) + (t2.v93-t1.v93)*(t2.v93-t1.v93) + (t2.v94-t1.v94)*(t2.v94-t1.v94) + (t2.v95-t1.v95)*(t2.v95-t1.v95) + (t2.v96-t1.v96)*(t2.v96-t1.v96) + (t2.v97-t1.v97)*(t2.v97-t1.v97) + (t2.v98-t1.v98)*(t2.v98-t1.v98) + (t2.v99-t1.v99)*(t2.v99-t1.v99) + (t2.v100-t1.v100)*(t2.v100-t1.v100) + (t2.v101-t1.v101)*(t2.v101-t1.v101) + (t2.v102-t1.v102)*(t2.v102-t1.v102) + (t2.v103-t1.v103)*(t2.v103-t1.v103) + (t2.v104-t1.v104)*(t2.v104-t1.v104) + (t2.v105-t1.v105)*(t2.v105-t1.v105) + (t2.v106-t1.v106)*(t2.v106-t1.v106) + (t2.v107-t1.v107)*(t2.v107-t1.v107) + (t2.v108-t1.v108)*(t2.v108-t1.v108) + (t2.v109-t1.v109)*(t2.v109-t1.v109) + (t2.v110-t1.v110)*(t2.v110-t1.v110) + (t2.v111-t1.v111)*(t2.v111-t1.v111) + (t2.v112-t1.v112)*(t2.v112-t1.v112) + (t2.v113-t1.v113)*(t2.v113-t1.v113) + (t2.v114-t1.v114)*(t2.v114-t1.v114) + (t2.v115-t1.v115)*(t2.v115-t1.v115) + (t2.v116-t1.v116)*(t2.v116-t1.v116) + (t2.v117-t1.v117)*(t2.v117-t1.v117) + (t2.v118-t1.v118)*(t2.v118-t1.v118) + (t2.v119-t1.v119)*(t2.v119-t1.v119) + (t2.v120-t1.v120)*(t2.v120-t1.v120) + (t2.v121-t1.v121)*(t2.v121-t1.v121) + (t2.v122-t1.v122)*(t2.v122-t1.v122) + (t2.v123-t1.v123)*(t2.v123-t1.v123) + (t2.v124-t1.v124)*(t2.v124-t1.v124) + (t2.v125-t1.v125)*(t2.v125-t1.v125) + (t2.v126-t1.v126)*(t2.v126-t1.v126) + (t2.v127-t1.v127)*(t2.v127-t1.v127)"
)

type AssetDeleteRequest struct {
	IDs []uint64 `json:"ids" binding:"required"`
}

type AssetFavouriteRequest struct {
	ID           uint64 `json:"id" binding:"required"`
	AlbumAssetID uint64 `json:"album_asset_id"`
}

// TODO: Move to before save in Asset
func GetTypeFrom(mimeType string) uint {
	if strings.HasPrefix(mimeType, "image/") {
		return models.AssetTypeImage
	}
	if strings.HasPrefix(mimeType, "video/") {
		return models.AssetTypeVideo
	}
	return models.AssetTypeOther
}

func LoadAssetsFromRows(c *gin.Context, rows *sql.Rows) *[]AssetInfo {
	result := []AssetInfo{}
	mimeType := ""
	for rows.Next() {
		assetInfo := AssetInfo{}
		if err := rows.Scan(&assetInfo.ID, &assetInfo.Name, &assetInfo.Owner, &assetInfo.Created, &assetInfo.DID, &mimeType,
			&assetInfo.GpsLat, &assetInfo.GpsLong, &assetInfo.Location, &assetInfo.Size, &assetInfo.MimeType, &assetInfo.Favourite); err != nil {

			log.Printf("DB error: %v", err)
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return nil
		}
		assetInfo.Type = GetTypeFrom(mimeType)
		result = append(result, assetInfo)
	}
	return &result
}

func AssetList(c *gin.Context, user *models.User) {
	fr := AssetsForFaceRequest{}
	_ = c.ShouldBindQuery(&fr)

	// Modified depends on deleted assets as well, that's why the where condition is different
	tx := db.Instance.
		Table("assets").
		Select("max(updated_at)").
		Where("user_id=? AND size>0 AND thumb_size>0", user.ID)
	if fr.FaceID == 0 && isNotModified(c, tx) {
		return
	}
	// TODO: For big sets maybe dynamically load asset info individually?
	tmp := db.Instance.
		Table("assets").
		Select(AssetsSelectClause).
		Joins("left join favourite_assets on favourite_assets.asset_id = assets.id").
		Joins(LeftJoinForLocations)
	if fr.FaceID > 0 {
		tmp = tmp.Joins("join (select distinct t2.asset_id from faces t1 join faces t2 where t1.id=? and "+vectorDotProduct+" <= ?) f on f.asset_id = assets.id", fr.FaceID, fr.Threshold)
	}
	rows, err := tmp.
		Where("assets.user_id=? and assets.deleted=0 and assets.size>0 and assets.thumb_size>0", user.ID).Order("assets.created_at DESC").Rows()
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

func AssetFetch(c *gin.Context, user *models.User) {
	RealAssetFetch(c, user.ID)
}

func checkAlbumAccess(c *gin.Context, checkUser, assetID uint64) bool {
	// Check if we have access via any shared album or if any of those albums is ours
	var sum int64
	result := db.Instance.Raw("select sum(ifnull(album_contributors.user_id, ifnull(albums.user_id, 0))) "+
		"from album_assets "+
		"left join album_contributors on (album_contributors.album_id = album_assets.album_id and album_contributors.user_id = ?) "+
		"left join albums on (albums.id = album_assets.album_id and albums.user_id = ?) "+
		"where asset_id=?", checkUser, checkUser, assetID).Scan(&sum)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return false
	}
	if sum == 0 {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return false
	}
	return true
}

func RealAssetFetch(c *gin.Context, checkUser uint64) {
	r := AssetFetchRequest{}
	err := c.ShouldBindQuery(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{
		ID: r.ID,
	}
	db.Instance.Joins("Bucket").First(&asset)
	if checkUser > 0 && asset.UserID != checkUser {
		if !checkAlbumAccess(c, checkUser, r.ID) {
			return
		}
	}
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		panic("Storage is nil")
	}
	if asset.Bucket.IsS3() {
		isThumb := false
		if r.Thumb == 1 && asset.ThumbSize > 0 {
			isThumb = true
		}
		// Redirect to the S3 location
		url, expires := asset.GetS3DownloadURL(isThumb)
		maxAge := expires - time.Now().Unix()
		c.Header("cache-control", "private, max-age="+strconv.FormatInt(maxAge, 10))
		c.Redirect(302, url)
		return
	}
	c.Header("cache-control", "private, max-age=604800")
	if r.Thumb == 1 && asset.ThumbSize > 0 {
		c.Header("content-type", "image/jpeg")
		if r.Size == 0 {
			// Default big (1280) thumb size
			_, err = storage.Load(asset.ThumbPath, c.Writer)
		} else {
			// Custom size
			var buf bytes.Buffer
			if _, err = storage.Load(asset.ThumbPath, &buf); err == nil {
				var imageThumbInfo utils.ImageThumbConverted
				imageThumbInfo, err = utils.CreateThumb(r.Size, &buf, c.Writer)
				c.Header("content-length", strconv.FormatInt(imageThumbInfo.ThumbSize, 10))
			}
		}
	} else {
		// Original
		c.Header("content-type", asset.MimeType)
		if r.Download == 1 {
			c.Header("content-disposition", "attachment; filename=\""+asset.Name+"\"")
		}
		// Handles Byte-ranges too
		storage.Serve(asset.Path, c.Request, c.Writer)
		return
	}
	// Handle errors
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
	}
}

func AssetDelete(c *gin.Context, user *models.User) {
	r := AssetDeleteRequest{}
	err := c.ShouldBindWith(&r, binding.JSON)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	failed := []uint64{}
	for _, id := range r.IDs {
		asset := models.Asset{
			ID: id,
		}
		db.Instance.Joins("Bucket").First(&asset)
		if asset.ID != id || asset.UserID != user.ID {
			failed = append(failed, id)
			log.Printf("Asset: %d, auth error", id)
			continue
		}
		asset.Deleted = true
		err = db.Instance.Save(&asset).Error
		if err != nil {
			failed = append(failed, id)
			log.Printf("Asset: %d, save error %s", id, err)
			continue
		}
		// TODO: Delete record better (and rely on cascaded deletes) and reinsert with same RemoteID (to stop re-uploading)?
		db.Instance.Exec("delete from album_assets where asset_id=?", id)
		db.Instance.Exec("delete from favourite_assets where asset_id=?", id)
		storage := storage.StorageFrom(&asset.Bucket)
		if storage == nil {
			log.Printf("Asset: %d, error: storage is nil", id)
			failed = append(failed, id)
			continue
		}
		// Finally delete
		if err = storage.Delete(asset.ThumbPath); err != nil {
			log.Printf("Asset: %d, thumb delete error: %s", id, err.Error())
		}
		if err = storage.Delete(asset.Path); err != nil {
			log.Printf("Asset: %d, delete error: %s", id, err.Error())
		}
		// Remote (S3) as well
		if err = storage.DeleteRemoteFile(asset.ThumbPath); err != nil {
			log.Printf("Remote Asset: %d, thumb delete error: %s", id, err.Error())
		}
		if err = storage.DeleteRemoteFile(asset.Path); err != nil {
			log.Printf("Remote Asset: %d, delete error: %s", id, err.Error())
		}
	}
	// Handle errors
	if len(failed) > 0 {
		c.JSON(http.StatusInternalServerError, MultiResponse{"Some assets cannot be deleted", failed})
		return
	}
	c.JSON(http.StatusOK, OKMultiResponse)
}

func AssetFavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	asset := models.Asset{ID: r.ID}
	db.Instance.First(&asset)
	if asset.ID != r.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	if r.AlbumAssetID == 0 || asset.UserID == user.ID {
		r.AlbumAssetID = 0
		// This must be our own asset
		if asset.ID != r.ID || asset.UserID != user.ID {
			c.JSON(http.StatusUnauthorized, Nope2Response)
			return
		}
	} else {
		// We should have access to this album
		albumAsset := models.AlbumAsset{ID: r.AlbumAssetID}
		db.Instance.First(&albumAsset)
		if albumAsset.ID != r.AlbumAssetID || albumAsset.AssetID != r.ID {
			c.JSON(http.StatusUnauthorized, Nope3Response)
			return
		}
		if !checkAlbumAccess(c, user.ID, r.ID) {
			return
		}
	}
	// All checks done! Phew...
	fav := models.FavouriteAsset{
		UserID:       user.ID,
		AssetID:      r.ID,
		AlbumAssetID: nil,
	}
	if r.AlbumAssetID > 0 {
		fav.AlbumAssetID = &r.AlbumAssetID
	}
	if db.Instance.Create(&fav).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}

func AssetUnfavourite(c *gin.Context, user *models.User) {
	r := AssetFavouriteRequest{}
	err := c.ShouldBindJSON(&r)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{err.Error()})
		return
	}
	fav := models.FavouriteAsset{}
	err = db.Instance.First(&fav, "user_id=? AND asset_id=?", user.ID, r.ID).Error
	if err != nil || fav.UserID != user.ID || fav.AssetID != r.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	if db.Instance.Delete(&fav).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError3Response)
		return
	}
	c.JSON(http.StatusOK, OKResponse)
}
