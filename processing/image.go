package processing

import (
	"bytes"
	"image"
	"log"
	"math"
	"os/exec"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strings"
	"time"
)

func getNextImageForProcessing(lastProcessedID uint64) (result models.Asset) {
	db.Instance.Where("deleted=0 AND size>0 AND mime_type LIKE 'image/%' AND unix_timestamp()-assets.created_at>30 AND assets.id>? AND "+
		"(width=0 OR height=0 OR thumb_size=0)",

		lastProcessedID).Order("id ASC").Limit(1).Joins("Bucket").Find(&result)
	return
}

// processOneImage return the asset.ID on success and 0 on error
func processOneImage(asset *models.Asset) uint64 {
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		log.Println("image-processing.go, StartProcessing: Storage is nil")
		return 0
	}
	if err := storage.EnsureLocalFile(asset.GetPath()); err != nil {
		log.Printf("Error downloading file from S3 for %s: %s\n", asset.GetPath(), err)
		return 0
	}
	defer storage.ReleaseLocalFile(asset.GetPath())

	// TODO: merge with video??
	// We need to get EXIF metadata?
	if asset.Width == 0 || asset.Height == 0 {
		cmd := exec.Command("exiftool", "-n", "-T", "-gpslatitude", "-gpslongitude", "-imagewidth", "-imageheight", "-duration", storage.GetFullPath(asset.GetPath()))
		output, err := cmd.Output()
		if err == nil {
			result := strings.Split(strings.Trim(string(output), "\n\t\r "), "\t")
			log.Printf("%+v", result)
			if len(result) == 5 {
				if result[0] != "-" {
					asset.GpsLat = utils.StringToFloat64Ptr(result[0])
				}
				if result[1] != "-" {
					asset.GpsLong = utils.StringToFloat64Ptr(result[1])
				}
				if result[2] != "-" {
					asset.Width = utils.StringToUInt16(result[2])
				}
				if result[3] != "-" {
					asset.Height = utils.StringToUInt16(result[3])
				}
				if result[4] != "-" {
					d := utils.StringToFloat64Ptr(result[4])
					asset.Duration = uint32(math.Ceil(*d))
				}
			}
			// finally - save it
			db.Instance.Save(&asset)
		}
	}
	// TODO: this doesn't support HEIF (HEIC)?
	// NOTE: files uploaded by browser should be covered (JPEGs)
	// Create thumbnail if missing
	if asset.ThumbSize == 0 {
		cmd := exec.Command("ffmpeg", "-y", "-i", storage.GetFullPath(asset.GetPath()), "-vf", "scale=min(1280\\,iw):-1", "-ss", "00:00:00.000", "-vframes", "1", storage.GetFullPath(asset.GetThumbPath()))
		err := cmd.Run()
		if err != nil {
			log.Printf("Error creating thumbnail for %s: %s", asset.GetPath(), err.Error())
		} else {
			buf := bytes.Buffer{}
			storage.Load(asset.GetThumbPath(), &buf)
			asset.ThumbSize = int64(buf.Len())
			thumb, _, err := image.Decode(&buf)
			if err != nil {
				log.Print("Error reading thumbnail for " + asset.GetThumbPath() + ": " + err.Error())
			} else {
				defer storage.ReleaseLocalFile(asset.GetThumbPath())

				asset.ThumbWidth = uint16(thumb.Bounds().Dx())
				asset.ThumbHeight = uint16(thumb.Bounds().Dy())
				asset.MimeType = "image/jpeg"
				db.Instance.Save(&asset)
				err := storage.UpdateFile(asset.GetThumbPath(), asset.MimeType)
				if err != nil {
					log.Println(err.Error())
					return 0
				}
			}
		}
	}
	return asset.ID
}

func StartProcessingImages() {
	lastProcessedID := uint64(0)
	for {
		asset := getNextImageForProcessing(lastProcessedID)
		if asset.ID == 0 {
			// Nothing to process...
			time.Sleep(30 * time.Second)
			lastProcessedID = 0
			continue
		}
		lastProcessedID = processOneImage(&asset)
		if lastProcessedID == 0 {
			// An error occurred, wait a bit
			time.Sleep(30 * time.Second)
		}
	}
}
