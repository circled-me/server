package processing

import (
	"log"
	"math"
	"os/exec"
	"server/db"
	"server/models"
	"server/storage"
	"server/utils"
	"strings"
)

type metadata struct{}

func (md *metadata) shouldHandle(asset *models.Asset) bool {
	return asset.Width == 0 || asset.Height == 0 || (asset.Duration == 0 && asset.IsVideo())
}

func (md *metadata) requiresContent(asset *models.Asset) bool {
	return true
}

func (md *metadata) process(asset *models.Asset, storage storage.StorageAPI) (int, func()) {
	cmd := exec.Command("exiftool", "-n", "-T", "-gpslatitude", "-gpslongitude", "-imagewidth", "-imageheight", "-duration", storage.GetFullPath(asset.GetPath()))
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Metadata processing error: %v", err)
		return Failed, nil
	}
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
	if err = db.Instance.Save(&asset).Error; err != nil {
		log.Printf("Error updating DB for asset ID %d: %v", asset.ID, err)
		return FailedDB, nil
	}
	return Done, nil
}
