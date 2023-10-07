package processing

import (
	"log"
	"math"
	"os/exec"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strconv"
	"strings"
	"time"

	"github.com/zsefvlol/timezonemapper"
)

type metadata struct{}

func (md *metadata) shouldHandle(asset *models.Asset) bool {
	return asset.Width == 0 ||
		asset.Height == 0 ||
		(asset.Duration == 0 && asset.IsVideo()) ||
		asset.UpdatedAt-asset.CreatedAt < 60 ||
		asset.TimeOffset == nil
}

func (md *metadata) requiresContent(asset *models.Asset) bool {
	return true
}

func (md *metadata) process(asset *models.Asset, storage storage.StorageAPI) (int, func()) {
	cmd := exec.Command("exiftool", "-n", "-T", "-gpslatitude", "-gpslongitude", "-imagewidth", "-imageheight", "-duration", "-createdate", "-offsettime", storage.GetFullPath(asset.Path))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Metadata processing error: %v", err)
		return Failed, nil
	}
	result := strings.Split(strings.Trim(string(output), "\n\t\r "), "\t")
	if len(result) == 7 {
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
		if result[6] != "-" {
			asset.TimeOffset = getTimeOffsetFrom(result[6])
		}
		// Still not having the time offset, but we have the GPS coordinates?
		if asset.TimeOffset == nil && asset.GpsLat != nil && asset.GpsLong != nil {
			zone, err := time.LoadLocation(timezonemapper.LatLngToTimezoneString(*asset.GpsLat, *asset.GpsLong))
			if err == nil && zone != nil {
				_, offset := time.Now().In(zone).Zone()
				log.Print(offset)
				asset.TimeOffset = &offset
			}
		}
		if result[5] != "-" {
			if t, err := time.Parse("2006:01:02 15:04:05", result[5]); err == nil {
				asset.CreatedAt = t.Unix()
				if asset.TimeOffset != nil {
					asset.CreatedAt -= int64(*asset.TimeOffset)
				}
			}
		}
	}
	if err = db.Instance.Save(&asset).Error; err != nil {
		log.Printf("Error updating DB for asset ID %d: %v", asset.ID, err)
		return FailedDB, nil
	}
	return Done, nil
}

// getTimeOffsetFrom return offset in seconds (or nil on error), input format is "+09:00"
func getTimeOffsetFrom(s string) *int {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return nil
	}
	hours, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil
	}
	mins, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil
	}
	result := hours * 3600
	if hours < 0 {
		result -= mins * 60
	} else {
		result += mins * 60
	}
	return &result
}
