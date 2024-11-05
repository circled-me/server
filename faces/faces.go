package faces

import (
	"log"
	"path/filepath"
	"server/config"

	"github.com/Kagami/go-face"
)

var (
	modelsDir = filepath.Join(".", "models")
	rec       *face.Recognizer
)

func init() {
	log.Println("Loading face recognition models...")
	// Init the recognizer.
	var err error
	rec, err = face.NewRecognizer(modelsDir)
	if err != nil {
		log.Fatalf("Can't init face recognizer: %v", err)
	}
}

func Detect(imgPath string) ([]face.Face, error) {
	log.Printf("Detecting faces in %s", imgPath)
	// Recognize faces on that image.
	if !config.FACEDETECT_CNN {
		return rec.RecognizeFile(imgPath)
	}
	// Default is CNN (Convolutional Neural Network) based detection
	return rec.RecognizeFileCNN(imgPath)
}