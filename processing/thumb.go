package processing

import (
	"bytes"
	"image"
	"log"
	"os/exec"
	"server/db"
	"server/models"
	"server/storage"
)

type thumb struct{}

func (t *thumb) shouldHandle(asset *models.Asset) bool {
	return asset.ThumbSize == 0
}

func (t *thumb) requiresContent(asset *models.Asset) bool {
	return true
}

func (t *thumb) process(asset *models.Asset, storage storage.StorageAPI) (status int, clean func()) {
	thumbPath := asset.CreateThumbPath()
	cmd := exec.Command("ffmpeg", "-y", "-i", storage.GetFullPath(asset.Path), "-vf", "scale=min(1280\\,iw):-1", "-ss", "00:00:00.000", "-vframes", "1", storage.GetFullPath(thumbPath))
	err := cmd.Run()
	if err != nil {
		log.Printf("Error creating thumbnail for asset %d, path:%s: %s", asset.ID, thumbPath, err.Error())
		return Failed, nil
	}
	buf := bytes.Buffer{}
	if _, err = storage.Load(thumbPath, &buf); err != nil {
		log.Printf("Cannot load newly created thumbnail for asset ID %d (%s) : %v", asset.ID, thumbPath, err)
		return Failed, nil
	}
	// Remove the temporary local file (in case of remote storage)
	clean = func() {
		storage.ReleaseLocalFile(thumbPath)
	}

	asset.ThumbSize = int64(buf.Len())
	thumb, _, err := image.Decode(&buf)
	if err != nil {
		log.Printf("Error decoding thumbnail for ID %d (%s): %v", asset.ID, thumbPath, err)
		return Failed, clean
	}
	asset.ThumbPath = thumbPath
	asset.ThumbWidth = uint16(thumb.Bounds().Dx())
	asset.ThumbHeight = uint16(thumb.Bounds().Dy())
	asset.PresignedThumbUntil = 0 // Clear S3 URL cache
	if err = db.Instance.Save(&asset).Error; err != nil {
		log.Printf("Error saving asset to DB for ID %d: %v", asset.ID, err)
		return Failed, clean
	}
	if err = storage.UpdateRemoteFile(asset.ThumbPath, asset.MimeType); err != nil {
		asset.ThumbSize = 0 // Revert
		asset.ThumbPath = ""
		db.Instance.Save(&asset)
		log.Printf("Error in storage.UpdateFile for asset ID %d (%s): %v", asset.ID, thumbPath, err)
		return Failed, nil
	}
	return Done, clean
}
