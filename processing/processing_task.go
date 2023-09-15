package processing

import (
	"log"
	"server/models"
	"strconv"
	"strings"
)

const (
	Skipped       = 0
	UserSkipped   = 1
	Done          = 2
	Failed        = 3
	FailedStorage = 4
)

type ProcessingTask struct {
	AssetID uint64       `gorm:"primaryKey"`
	Asset   models.Asset `gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	Status  string       `gorm:"type:varchar(1024)"` // Contains comma-separated pairs of integration and status, e.g. "video:1,thumb:2,another:0"
}

func (pt *ProcessingTask) statusToMap() map[string]int {
	result := map[string]int{}
	if pt.Status == "" {
		return result
	}
	for _, v := range strings.Split(pt.Status, ",") {
		current := strings.Split(v, ":")
		if len(current) != 2 {
			log.Printf("Task status contains invalid chars, asset: %d, status: %s", pt.AssetID, pt.Status)
			continue
		}
		result[current[0]], _ = strconv.Atoi(current[1])
	}
	return result
}

func (pt *ProcessingTask) updateWith(statusMap map[string]int) {
	result := []string{}
	for k, v := range statusMap {
		result = append(result, k+":"+strconv.Itoa(v))
	}
	pt.Status = strings.Join(result, ",")
}
