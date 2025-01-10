package handlers

import (
	"net/http"
	"server/config"
	"server/db"
	"server/models"
	"strconv"
	"strings"

	_ "image/jpeg"

	"github.com/gin-gonic/gin"
)

type FaceInfo struct {
	ID         uint64 `json:"id"`
	Num        int    `json:"num"`
	PersonID   uint64 `json:"person_id"`
	PersonName string `json:"person_name"`
	AsselID    uint64 `json:"asset_id"`
	X1         int    `json:"x1"`
	Y1         int    `json:"y1"`
	X2         int    `json:"x2"`
	Y2         int    `json:"y2"`
}

type AssetsForFaceRequest struct {
	FaceID    uint64  `form:"face_id" binding:"required"`
	Threshold float64 `form:"threshold"`
}

func FacesForAsset(c *gin.Context, user *models.User) {
	assetIDSt, exists := c.GetQuery("asset_id")
	if !exists {
		c.JSON(http.StatusBadRequest, Response{"Missing asset ID"})
		return
	}
	assetID, err := strconv.ParseUint(assetIDSt, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{"Invalid asset ID"})
		return
	}
	asset := models.Asset{ID: assetID}
	db.Instance.First(&asset)
	if asset.ID != assetID || asset.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	rows, err := db.Instance.Raw("select f.id, f.num, f.x1, f.y1, f.x2, f.y2, p.id, p.name from faces f left join people p on f.person_id=p.id where f.asset_id=?", assetID).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	defer rows.Close()
	result := []FaceInfo{}
	for rows.Next() {
		face := FaceInfo{}
		pID := &face.PersonID
		pName := &face.PersonName
		if err = rows.Scan(&face.ID, &face.Num, &face.X1, &face.Y1, &face.X2, &face.Y2, &pID, &pName); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			return
		}
		if pID != nil {
			face.PersonID = *pID
		}
		if pName != nil {
			face.PersonName = *pName
		}
		result = append(result, face)
	}
	c.JSON(http.StatusOK, result)
}

func PeopleList(c *gin.Context, user *models.User) {
	// Do this in two steps. First load all people information
	rows, err := db.Instance.Raw("select id, name from people where user_id=?", user.ID).Rows()
	if err != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// Put the info in the FaceInfo struct
	people := []FaceInfo{}
	for rows.Next() {
		person := FaceInfo{}
		if err = rows.Scan(&person.PersonID, &person.PersonName); err != nil {
			c.JSON(http.StatusInternalServerError, DBError2Response)
			rows.Close()
			return
		}
		people = append(people, person)
	}
	rows.Close()

	// Now load the last face for each person
	for i, person := range people {
		rows, err = db.Instance.Raw("select id, asset_id, num, x1, y1, x2, y2 from faces where person_id=? order by created_at desc limit 1", person.PersonID).Rows()
		if err != nil {
			c.JSON(http.StatusInternalServerError, DBError3Response)
			return
		}
		if rows.Next() {
			face := &people[i]
			if err = rows.Scan(&face.ID, &face.AsselID, &face.Num, &face.X1, &face.Y1, &face.X2, &face.Y2); err != nil {
				c.JSON(http.StatusInternalServerError, DBError4Response)
				rows.Close()
				return
			}
		}
		rows.Close()
	}
	c.JSON(http.StatusOK, people)
}

func CreatePerson(c *gin.Context, user *models.User) {
	var personFace FaceInfo
	err := c.ShouldBindJSON(&personFace)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
	personFace.PersonName = strings.Trim(personFace.PersonName, " ")
	if personFace.PersonName == "" {
		c.JSON(http.StatusBadRequest, Response{"Empty person name"})
		return
	}
	personModel := models.Person{Name: personFace.PersonName, UserID: user.ID}
	if db.Instance.Create(&personModel).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	personFace.PersonID = personModel.ID
	c.JSON(http.StatusOK, personFace)
}

func PersonAssignFace(c *gin.Context, user *models.User) {
	var face FaceInfo
	err := c.ShouldBindJSON(&face)
	if err != nil {
		c.JSON(http.StatusInternalServerError, Response{err.Error()})
		return
	}
	if face.PersonID == 0 {
		if face.ID == 0 {
			c.JSON(http.StatusBadRequest, Response{"Empty face ID and person ID"})
			return
		}
		// We want to unassign a face from a person
		if db.Instance.Exec("update faces set person_id=null where id=?", face.ID).Error != nil {
			c.JSON(http.StatusInternalServerError, DBError1Response)
			return
		}
		face.PersonID = 0
		face.PersonName = ""
		c.JSON(http.StatusOK, face)
		return
	}
	// Check if this face.PersonID is the same as current user.ID
	person := models.Person{ID: face.PersonID}
	if db.Instance.First(&person).Error != nil || person.UserID != user.ID {
		c.JSON(http.StatusUnauthorized, NopeResponse)
		return
	}
	if db.Instance.Exec("update faces set person_id=? where id=?", face.PersonID, face.ID).Error != nil {
		c.JSON(http.StatusInternalServerError, DBError1Response)
		return
	}
	// threshold is squared by default
	thresholdStr := c.Query("threshold")
	threshold, _ := strconv.ParseFloat(thresholdStr, 64)
	if threshold == 0 {
		threshold = config.FACE_MAX_DISTANCE_SQ
	}
	// Set PersonID to all assets with faces similar to the given face based on threshold
	// Also, make sure the distance is greater than the current face's distance (i.e. the new face is more similar to the one detected before)
	if db.Instance.Exec(`update faces 
						set person_id=? 
						where id in (
							select id from (
								select t2.id 
								from faces t1 join faces t2 
								where t1.id=? and 
									t1.id!=t2.id and 
									t1.user_id=? and 
									t1.user_id=t2.user_id and 
									`+models.FacesVectorDistance+` <= ? and 
									(t2.distance = 0 OR t2.distance > `+models.FacesVectorDistance+`)
							) tmp
						)`,
		face.PersonID, face.ID, user.ID, threshold).Error != nil {

		c.JSON(http.StatusInternalServerError, DBError2Response)
		return
	}
	face.PersonName = person.Name
	c.JSON(http.StatusOK, face)
}
