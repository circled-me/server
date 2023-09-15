package processing

import (
	"fmt"
	"log"
	"reflect"
	"server/db"
	"server/models"
	"server/storage"
	"sort"
	"time"
)

const (
	orderSooner = -10
	orderNormal = 0
	orderLater  = 10
)

type processingTask interface {
	order() int
	shouldHandle(*models.Asset) bool
	requiresContent(*models.Asset) bool // This method is necessary to establish if we need to download remote file contents
	process(*models.Asset, storage.StorageAPI) (status int, cleanup func())
}

type processingTasksElement struct {
	name string
	task processingTask
}

type processingTasks []processingTasksElement

var tasks = processingTasks{}

func (ts *processingTasks) register(t processingTask) {
	*ts = append(*ts, processingTasksElement{
		name: reflect.TypeOf(t).Elem().Name(),
		task: t,
	})
	sort.Slice(*ts, func(i, j int) bool {
		return (*ts)[i].task.order() < (*ts)[j].task.order()
	})
}

func (ts *processingTasks) requireContent(asset *models.Asset) bool {
	for _, e := range *ts {
		if e.task.requiresContent(asset) && e.task.shouldHandle(asset) {
			return true
		}
	}
	return false
}

func (ts *processingTasks) process(asset *models.Asset, assetStorage storage.StorageAPI, statusMap map[string]int) {
	// Cleanup tasks for the current asset
	cleanAll := []func(){}
	for _, e := range *ts {
		if _, ok := statusMap[e.name]; ok {
			// For now - just one try for each task
			continue
		}
		if !e.task.shouldHandle(asset) {
			statusMap[e.name] = Skipped
			continue
		}
		if e.task.requiresContent(asset) && assetStorage == nil {
			statusMap[e.name] = FailedStorage
			continue
		}
		start := time.Now()
		status, cleanup := e.task.process(asset, assetStorage)
		timeConsumed := time.Since(start).Milliseconds()

		statusMap[e.name] = status
		if cleanup != nil {
			cleanAll = append(cleanAll, cleanup)
		}
		log.Printf("Task %s, asset: %d, result: %d, time: %v", e.name, asset.ID, statusMap[e.name], timeConsumed)
	}
	for _, clean := range cleanAll {
		clean()
	}
}

func Init() {
	if err := db.Instance.AutoMigrate(&ProcessingTask{}); err != nil {
		log.Printf("Auto-migrate error: %v", err)
	}
	// Initialise all processing tasks
	tasks.register(&videoConvert{})
	tasks.register(&metadata{})
	tasks.register(&thumb{})
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
			log.Printf("processPending load asset error: %v, asset: %d", err, asset.ID)
			break
		}
		current := ProcessingTask{
			AssetID: asset.ID,
			Status:  status,
		}
		var assetStorage storage.StorageAPI
		if tasks.requireContent(&asset) {
			// Ensure we actually have access to the asset contents
			assetStorage = storage.StorageFrom(&asset.Bucket)
			if assetStorage == nil {
				fmt.Printf("processPending: Storage is nil for asset ID: %d", asset.ID)
			} else {
				if err = assetStorage.EnsureLocalFile(asset.GetPath()); err != nil {
					fmt.Printf("Error downloading remote file for %s: %v\n", asset.GetPath(), err)
				} else {
					// In the end - cleanup local copy
					defer assetStorage.ReleaseLocalFile(asset.GetPath())
				}
			}
		}
		statusMap := current.statusToMap()
		tasks.process(&asset, assetStorage, statusMap)
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

func StartProcessing() {
	for {
		processPending()
		time.Sleep(10 * time.Second)
	}
}
