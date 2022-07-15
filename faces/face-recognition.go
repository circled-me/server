package faces

import (
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

func ProcessPhoto(path string) ([]face.Face, error) {
	return recognizer.RecognizeFile(path)
}
