package faces

import (
	"server/models"
	"server/utils"

	"github.com/Kagami/go-face"
)

var (
	recognizer *face.Recognizer
)

func Init(modelsDir string) {
	var err error
	recognizer, err = face.NewRecognizer(modelsDir)
	if err != nil {
		// TODO: change this?
		panic(err)
	}
}

func ProcessPhoto(assetId uint64, path string) (foundFaces []models.Face, err error) {
	var f []face.Face
	f, err = recognizer.RecognizeFile(path)
	if err != nil {
		return
	}
	for _, cur := range f {
		desc := [128]float32(cur.Descriptor)
		foundFaces = append(foundFaces, models.Face{
			AssetID:    assetId,
			Descriptor: utils.Float32ArrayToByteArray(desc[:]),
			RectX1:     uint16(cur.Rectangle.Min.X),
			RectY1:     uint16(cur.Rectangle.Min.Y),
			RectX2:     uint16(cur.Rectangle.Max.X),
			RectY2:     uint16(cur.Rectangle.Max.Y),
		})
	}
	return
}
