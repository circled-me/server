package video

import (
	"bytes"
	"fmt"
	"image"
	"log"
	"math"
	"os/exec"
	"path/filepath"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strings"
	"time"
)

// convertVideo uses hard-coded options
func convertVideo(inFile, outFile string) error {
	cmd := exec.Command("ffmpeg", "-y", "-i", inFile, "-c:v", "libx264", "-c:a", "aac", "-b:a", "128k", "-crf", "24", outFile)
	return cmd.Run()
}

func getNextForProcessing(lastProcessedID uint64) (result models.Asset) {
	// select video assets that are not MP4 OR have been manually uploaded so don't have much meta data
	db.Instance.Where("deleted=0 AND size>0 AND mime_type LIKE 'video/%' AND unix_timestamp()-assets.created_at>30 AND assets.id>? AND "+
		"(mime_type!='video/mp4' OR width=0 OR height=0 OR thumb_width=0 OR thumb_height=0 OR thumb_size=0 OR duration=0)",

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
		// convert the video to mp4 if necessary
		if asset.MimeType != "video/mp4" {
			oldPath := asset.GetPath()
			ext := filepath.Ext(asset.Name)
			asset.Name = asset.Name[:len(asset.Name)-len(ext)] + ".mp4"
			err := convertVideo(storage.GetFullPath(oldPath), storage.GetFullPath(asset.GetPath()))
			if err == nil {
				log.Print("DONE video processing for:", asset.GetPath())
				asset.MimeType = "video/mp4"
				asset.Size = storage.GetSize(oldPath)
				if storage.Delete(oldPath) == nil && asset.Size > 0 {
					db.Instance.Save(&asset)
				}
			} else {
				fmt.Printf("ERROR in video processing for:%s, %v\n", asset.GetPath(), err)
			}
		}
		// we need to get video metadata?
		if asset.Width == 0 || asset.Height == 0 || asset.Duration == 0 {
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
					log.Printf("%+v", asset)
				}
				// finally - save it
				db.Instance.Save(&asset)
			}
		}
		// create thumbnail if missing
		if asset.ThumbHeight == 0 || asset.ThumbWidth == 0 || asset.ThumbSize == 0 {
			cmd := exec.Command("ffmpeg", "-y", "-i", storage.GetFullPath(asset.GetPath()), "-ss", "00:00:01.000", "-vframes", "1", storage.GetFullPath(asset.GetThumbPath()))
			err := cmd.Run()
			if err != nil {
				log.Print("Error creating thumbnail for "+asset.GetPath(), err.Error())
			} else {
				buf := bytes.Buffer{}
				storage.Load(asset.GetThumbPath(), &buf)
				asset.ThumbSize = int64(buf.Len())
				thumb, _, err := image.Decode(&buf)
				if err != nil {
					log.Print("Error reading thumbnail for " + asset.GetThumbPath() + ": " + err.Error())
				} else {
					asset.ThumbWidth = uint16(thumb.Bounds().Dx())
					asset.ThumbHeight = uint16(thumb.Bounds().Dy())
					db.Instance.Save(&asset)
				}
			}
		}
		lastProcessedID = asset.ID
	}
}
