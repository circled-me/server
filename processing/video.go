package processing

import (
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"server/db"
	"server/models"
	"server/storage"
)

const taskVideoConvert = "video_convert"

type videoConvert struct{}

func (vc *videoConvert) getName() string {
	return taskVideoConvert
}

func (vc *videoConvert) shouldHandle(asset *models.Asset) bool {
	return asset.IsVideo()
}

func (vc *videoConvert) process(asset *models.Asset) int {
	if asset.User.VideoSetting == models.VideoSettingSkip {
		return UserSkipped
	}
	if asset.MimeType == "video/mp4" {
		return Skipped
	}
	storage := storage.StorageFrom(&asset.Bucket)
	oldPath := asset.GetPath()
	ext := filepath.Ext(asset.Name)
	asset.Name = asset.Name[:len(asset.Name)-len(ext)] + ".mp4"
	newPath := asset.GetPath()
	err := ffmpeg(storage.GetFullPath(oldPath), storage.GetFullPath(newPath))
	asset.Size = storage.GetSize(newPath)
	// TODO: Is below correct??
	if err == nil || asset.Size <= 0 {
		log.Print("DONE video processing for:", asset.GetPath())

		defer storage.ReleaseLocalFile(newPath)
		asset.MimeType = "video/mp4"
		asset.PresignedUntil = 0
		err := storage.UpdateFile(newPath, asset.MimeType)
		if err != nil {
			log.Println(err.Error())
			return Failed
		}
		if db.Instance.Save(&asset).Error == nil {
			err1 := storage.DeleteRemoteFile(oldPath)
			err2 := storage.Delete(oldPath)
			if err1 != nil || err2 != nil {
				log.Printf("Error deleting object (remote,local): %v, %v", err1, err2)
			}
		}
	} else {
		fmt.Printf("ERROR in video processing for: %s, %v, size: %v\n", asset.GetPath(), err, asset.Size)
		return Failed
	}
	return Done
}

// ffmpeg uses hard-coded options
func ffmpeg(inFile, outFile string) error {
	log.Printf("Converting file %s to %s", inFile, outFile)
	cmd := exec.Command("ffmpeg", "-y", "-i", inFile, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k", "-crf", "24", outFile)
	return cmd.Run()
}
