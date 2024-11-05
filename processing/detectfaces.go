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
	for i, face := range result {
		faceModel := models.Face{
			AssetID: asset.ID,
			Num:     i,
			X1:      face.Rectangle.Min.X,
			Y1:      face.Rectangle.Min.Y,
			X2:      face.Rectangle.Max.X,
			Y2:      face.Rectangle.Max.Y,
		}
		// Use reflection to set Vx fields to the corresponding value from the array
		for j, value := range face.Descriptor {
			reflect.ValueOf(&faceModel).Elem().FieldByName("V" + strconv.Itoa(j)).SetFloat(float64(value))
		}
		if err := db.Instance.Create(&faceModel).Error; err != nil {
			log.Printf("Error saving face location for asset %d: %v", asset.ID, err)
			return Failed, nil
		}
	}
	return Done, clean
}
