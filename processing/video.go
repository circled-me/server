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

type videoConvert struct{}

func (vc *videoConvert) shouldHandle(asset *models.Asset) bool {
	return asset.IsVideo() && asset.MimeType != "video/mp4"
}

func (vc *videoConvert) requiresContent(asset *models.Asset) bool {
	return true
}

func (vc *videoConvert) process(asset *models.Asset, storage storage.StorageAPI) (status int, clean func()) {
	if asset.User.VideoSetting == models.VideoSettingSkip {
		return UserSkipped, nil
	}
	oldPath := asset.GetPath()
	ext := filepath.Ext(asset.Name)
	asset.Name = asset.Name[:len(asset.Name)-len(ext)] + ".mp4"
	newPath := asset.GetPath()
	err := ffmpegConvert(storage.GetFullPath(oldPath), storage.GetFullPath(newPath))
	asset.Size = storage.GetSize(newPath)
	// Always cleanup in the end
	clean = func() {
		// Delete the temp file after all tasks have completed
		storage.ReleaseLocalFile(newPath)
	}

	if err != nil || asset.Size <= 0 {
		fmt.Printf("ERROR in video processing for: %s, %v, size: %v\n", asset.GetPath(), err, asset.Size)
		return Failed, clean
	}
	log.Print("DONE video processing for:", asset.GetPath())

	asset.MimeType = "video/mp4"
	asset.PresignedUntil = 0
	if err := storage.UpdateFile(newPath, asset.MimeType); err != nil {
		log.Printf("Error updating asset ID %d (%s->%s): %v", asset.ID, newPath, asset.GetPath(), err)
		return Failed, clean
	}
	if err = db.Instance.Save(&asset).Error; err != nil {
		log.Printf("Error updating DB for asset ID %d: %v", asset.ID, err)
		return Failed, clean
	}
	// Delete old files and objects
	err1 := storage.DeleteRemoteFile(oldPath)
	err2 := storage.Delete(oldPath)
	if err1 != nil || err2 != nil {
		log.Printf("Error deleting old objects for asset ID %d (%s), errors (remote,local): %v, %v", asset.ID, oldPath, err1, err2)
	}
	return Done, clean
}

// ffmpegConvert uses hard-coded options
func ffmpegConvert(inFile, outFile string) error {
	log.Printf("Converting file %s to %s", inFile, outFile)
	cmd := exec.Command("ffmpeg", "-y", "-i", inFile, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k", "-crf", "24", "-movflags", "use_metadata_tags", "-map_metadata", "0", outFile)
	return cmd.Run()
}
