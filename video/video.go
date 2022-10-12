package video

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"server/db"
	"server/models"
	"server/storage"
	"time"
)

// convertVideo uses hard-coded options
func convertVideo(inFile, outFile string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inFile, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k", "-crf", "24", outFile)
	return cmd.Run()
}

func getNextForProcessing(lastProcessedID uint64) (result models.Asset) {
	// select video assets that are not MP4
	db.Instance.Where("deleted=0 AND size > 0 AND mime_type LIKE 'video/%' AND mime_type != 'video/mp4' AND unix_timestamp()-assets.created_at>30 AND assets.id>?",
		lastProcessedID).Limit(1).Joins("Bucket").Find(&result)
	return
}

func StartProcessing() {
	lastProcessedID := uint64(0)
	for {
		asset := getNextForProcessing(lastProcessedID)
		if asset.ID == 0 {
			// Nothing to process...
			time.Sleep(30 * time.Second)
			lastProcessedID = 0
			continue
		}
		storage := storage.StorageFrom(&asset.Bucket)
		oldPath := asset.GetPath()
		ext := filepath.Ext(asset.Name)
		asset.Name = asset.Name[:len(asset.Name)-len(ext)-1] + ".mp4"
		err := convertVideo(storage.GetFullPath(oldPath), storage.GetFullPath(asset.GetPath()))
		if err == nil {
			fmt.Println("DONE video processing for:", asset.GetPath())
			asset.MimeType = "video/mp4"
			if storage.Delete(oldPath) == nil {
				db.Instance.Save(&asset)
			}
		} else {
			fmt.Printf("ERROR in video processing for:%s, %v\n", asset.GetPath(), err)
		}
		lastProcessedID = asset.ID
	}
}
