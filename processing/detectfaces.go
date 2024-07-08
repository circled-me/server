package processing

import (
	"log"
	"reflect"
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
	for i, face := range result.Locations {
		if i >= len(result.Encodings) {
			// There should be always a corresponding encoding for each face location
			log.Printf("Error: face location %d without encoding for asset %d", i, asset.ID)
			break
		}
		faceModel := models.Face{
			AssetID: asset.ID,
			Num:     i,
			Top:     face[faces.IndexTop],
			Right:   face[faces.IndexRight],
			Bottom:  face[faces.IndexBottom],
			Left:    face[faces.IndexLeft],
		}
		// Use reflection to set Vx fields to the corresponding value from the array
		for j, value := range result.Encodings[i] {
			reflect.ValueOf(&faceModel).Elem().FieldByName("V" + strconv.Itoa(j)).SetFloat(value)
		}
		if err := db.Instance.Create(&faceModel).Error; err != nil {
			log.Printf("Error saving face location for asset %d: %v", asset.ID, err)
			return Failed, nil
		}
	}
	return Done, clean
}
