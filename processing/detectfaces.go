package processing

import (
	"log"
	"reflect"
	"server/config"
	"server/db"
	"server/faces"
	"server/models"
	"server/storage"
	"strconv"
)

type detectfaces struct{}

func (t *detectfaces) shouldHandle(asset *models.Asset) bool {
	return true
}

func (t *detectfaces) requiresContent(asset *models.Asset) bool {
	return true
}

func (t *detectfaces) process(asset *models.Asset, storage storage.StorageAPI) (status int, clean func()) {

	if asset.ThumbPath == "" {
		return Failed, nil
	}
	if storage.GetSize(asset.ThumbPath) <= 0 {
		if storage.EnsureLocalFile(asset.ThumbPath) != nil {
			return Failed, nil
		}
	}
	clean = func() {
		storage.ReleaseLocalFile(asset.ThumbPath)
	}
	// Extract faces
	result, err := faces.Detect(storage.GetFullPath(asset.ThumbPath))
	if err != nil {
		log.Printf("Error detecting faces for asset %d, path:%s: %s", asset.ID, asset.ThumbPath, err.Error())
		return Failed, nil
	}
	// Save faces' data to DB
	for i, face := range result {
		faceModel := models.Face{
			AssetID:  asset.ID,
			Num:      i,
			X1:       face.Rectangle.Min.X,
			Y1:       face.Rectangle.Min.Y,
			X2:       face.Rectangle.Max.X,
			Y2:       face.Rectangle.Max.Y,
			PersonID: nil,
		}
		// Use reflection to set Vx fields to the corresponding value from the array
		for j, value := range face.Descriptor {
			reflect.ValueOf(&faceModel).Elem().FieldByName("V" + strconv.Itoa(j)).SetFloat(float64(value))
		}
		if err := db.Instance.Create(&faceModel).Error; err != nil {
			log.Printf("Error saving face location for asset %d: %v", asset.ID, err)
			return Failed, nil
		}
		// Find the face that is most similar (least distance) to this one and fetch it's person_id
		db.Instance.Raw(`select t2.person_id, `+models.FacesVectorDistance+` as threshold 
						 from faces t1 join faces t2 
						 where t1.id=? and t2.person_id is not null and t1.id != t2.id 
						 order by threshold limit 1`, faceModel.ID).Debug().Row().Scan(&faceModel.PersonID, &faceModel.Distance)
		log.Printf("Face %d, threshold: %f\n", faceModel.ID, faceModel.Distance)
		if faceModel.PersonID != nil && faceModel.Distance <= config.FACE_MAX_DISTANCE_SQ {
			// Update the current face with the found person_id
			db.Instance.Save(&faceModel)
			log.Printf("Updated face %d, person_id: %d\n", faceModel.ID, *faceModel.PersonID)
		}
	}
	return Done, clean
}
