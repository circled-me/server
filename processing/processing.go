package processing

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

type processingTask interface {
	getName() string
	process(*models.Asset) int
	shouldHandle(*models.Asset) bool
}

var (
	tasks = map[string]processingTask{}
)

func registerTask(t processingTask) {
	tasks[t.getName()] = t
}
func Init() {
	if err := db.Instance.AutoMigrate(&ProcessingTask{}); err != nil {
		log.Printf("Auto-migrate error: %v", err)
	}
	// Initialise all processing tasks
	registerTask(&videoConvert{})
}

// TODO: 2 or more in parallel? Depending on CPU count?
func processPending() {
	// All assets that don't hvave processing_tasks record, OR
	// status has fewer tasks performed than the currently available ones
	rows, err := db.Instance.
		Table("assets").
		Joins("LEFT JOIN processing_tasks ON (assets.id = processing_tasks.asset_id)").
		Select("assets.id, IFNULL(processing_tasks.status, ''), processing_tasks.asset_id").
		Where("assets.deleted=0 AND "+
			"assets.size>0 AND "+
			"unix_timestamp()-assets.created_at>30 AND "+
			"(processing_tasks.status IS NULL OR "+
			"  LENGTH(processing_tasks.status)-LENGTH(REPLACE(processing_tasks.status, ',', ''))+1 < ?)", len(tasks)).
		Order("assets.created_at").Rows()
	if err != nil {
		log.Printf("processPending error: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		asset := models.Asset{}
		status := ""
		var recordId *uint64
		if err = rows.Scan(&asset.ID, &status, &recordId); err != nil {
			log.Printf("processPending row error: %v", err)
			break
		}
		if err = db.Instance.Preload("Bucket").Preload("User").First(&asset).Error; err != nil {
			log.Printf("processPending load asset error: %v", err)
			continue
		}
		current := ProcessingTask{
			AssetID: asset.ID,
			Status:  status,
		}
		statusMap := current.statusToMap()
		for taskName, task := range tasks {
			if _, ok := statusMap[taskName]; ok {
				// TODO: Have retries maybe?
				// For now - just one try for each task
				continue
			}
			if !task.shouldHandle(&asset) {
				statusMap[taskName] = Skipped
				continue
			}
			start := time.Now()
			statusMap[taskName] = task.process(&asset)
			timeConsumed := time.Since(start).Milliseconds()
			log.Printf("Task %s, asset: %d, result: %d, time: %v", taskName, asset.ID, statusMap[taskName], timeConsumed)
		}
		current.updateWith(statusMap)
		if recordId == nil {
			// This is a new record
			err = db.Instance.Create(&current).Error
		} else {
			// This is an update
			err = db.Instance.Save(&current).Error
		}
		if err != nil {
			log.Printf("processPending save task error: %v", err)
		}
	}
}

func getNextForProcessing(lastProcessedID uint64) (result models.Asset) {
	// Select video assets that are not MP4 OR have been manually uploaded so don't have enough meta data
	videoCondition := "(  mime_type LIKE 'video/%' AND (mime_type!='video/mp4' OR width=0 OR height=0 OR thumb_size=0 OR duration=0)  )"
	// Select image assets that are don't have thumbnail or enough meta data
	imageCondition := "(  mime_type LIKE 'image/%' AND (width=0 OR height=0 OR thumb_size=0)  )"

	db.Instance.Where("deleted=0 AND size>0 AND unix_timestamp()-assets.created_at>30 AND assets.id>? AND "+
		"("+videoCondition+" OR "+imageCondition+" )",

		lastProcessedID).Order("id ASC").Limit(1).Joins("Bucket").Find(&result)
	return
}

// processOne return the asset.ID on success and 0 on error
func processOne(asset *models.Asset) uint64 {
	storage := storage.StorageFrom(&asset.Bucket)
	if storage == nil {
		log.Println("video-processing.go, StartProcessing: Storage is nil")
		return 0
	}
	if err := storage.EnsureLocalFile(asset.GetPath()); err != nil {
		log.Printf("Error downloading file from S3 for %s: %s\n", asset.GetPath(), err)
		return 0
	}
	defer storage.ReleaseLocalFile(asset.GetPath())

	isVideo := asset.IsVideo()
	// Convert the video to mp4 if necessary
	if isVideo && asset.MimeType != "video/mp4" {
		oldPath := asset.GetPath()
		ext := filepath.Ext(asset.Name)
		asset.Name = asset.Name[:len(asset.Name)-len(ext)] + ".mp4"
		newPath := asset.GetPath()
		err := ffmpeg(storage.GetFullPath(oldPath), storage.GetFullPath(newPath))
		asset.Size = storage.GetSize(newPath)
		if err == nil || asset.Size <= 0 {
			log.Print("DONE video processing for:", asset.GetPath())

			defer storage.ReleaseLocalFile(newPath)
			asset.MimeType = "video/mp4"
			asset.PresignedUntil = 0
			err := storage.UpdateFile(newPath, asset.MimeType)
			if err != nil {
				log.Println(err.Error())
				return 0
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
		}
	}
	// We need to get video metadata?
	if asset.Width == 0 || asset.Height == 0 || (isVideo && asset.Duration == 0) {
		cmd := exec.Command("exiftool", "-n", "-T", "-gpslatitude", "-gpslongitude", "-imagewidth", "-imageheight", "-duration", storage.GetFullPath(asset.GetPath()))
		output, err := cmd.Output()
		if err == nil {
			result := strings.Split(strings.Trim(string(output), "\n\t\r "), "\t")
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
	// Create thumbnail if missing
	if asset.ThumbSize == 0 {
		cmd := exec.Command("ffmpeg", "-y", "-i", storage.GetFullPath(asset.GetPath()), "-vf", "scale=min(1280\\,iw):-1", "-ss", "00:00:00.000", "-vframes", "1", storage.GetFullPath(asset.GetThumbPath()))
		err := cmd.Run()
		if err != nil {
			log.Printf("Error creating thumbnail for %s: %s", asset.GetPath(), err.Error())
		} else {
			buf := bytes.Buffer{}
			_, err := storage.Load(asset.GetThumbPath(), &buf)
			if err != nil {
				log.Printf("Cannot load object %v", err)
				return 0
			}
			asset.ThumbSize = int64(buf.Len())
			thumb, _, err := image.Decode(&buf)
			if err != nil {
				log.Print("Error reading thumbnail for " + asset.GetThumbPath() + ": " + err.Error())
			} else {
				defer storage.ReleaseLocalFile(asset.GetThumbPath())

				asset.ThumbWidth = uint16(thumb.Bounds().Dx())
				asset.ThumbHeight = uint16(thumb.Bounds().Dy())
				asset.MimeType = "image/jpeg"
				asset.PresignedThumbUntil = 0 // Clear S3 URL cache
				if db.Instance.Save(&asset).Error == nil {
					err := storage.UpdateFile(asset.GetThumbPath(), asset.MimeType)
					if err != nil {
						asset.ThumbSize = 0 // Revert
						db.Instance.Save(&asset)
						log.Println(err.Error())
						return 0
					}
				}
			}
		}
	}
	return asset.ID
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
		lastProcessedID = processOne(&asset)
		if lastProcessedID == 0 {
			// An error occurred, wait a bit
			time.Sleep(30 * time.Second)
		}
	}
	// for {
	// 	processPending()
	// 	time.Sleep(10 * time.Second)
	// }
}
