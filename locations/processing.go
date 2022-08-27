package locations

import (
	"fmt"
	"server/db"
	"server/models"
	"time"
)

func getNextForProcessing(lastProcessedID uint64) (result models.Asset) {
	db.Instance.Where("deleted=0 AND place_id IS NULL AND gps_lat IS NOT NULL AND gps_long IS NOT NULL AND unix_timestamp()-created_at>300 AND id>?",
		lastProcessedID).Limit(1).Find(&result)
	return
}

func StartProcessing() {
	lastProcessedID := uint64(0)
	for i := 0; i < 500; i++ {
		asset := getNextForProcessing(lastProcessedID)
		if asset.ID == 0 {
			// Nothing to process...
			time.Sleep(30 * time.Second)
			lastProcessedID = 0
			continue
		}
		_ = process(&asset)
		lastProcessedID = asset.ID
	}
}

func process(a *models.Asset) bool {
	// Truncate - only use 0.0001 of precision
	lat := float64(int(*a.GpsLat*10000)) / 10000
	long := float64(int(*a.GpsLong*10000)) / 10000
	// Try first local DB
	location := models.Location{
		GpsLat:  lat,
		GpsLong: long,
	}
	db.Instance.Limit(1).Find(&location, location)
	// fmt.Printf("Location found: %+v\n\n", location)
	a.PlaceID = location.GetPlaceID()
	if a.PlaceID > 0 {
		fmt.Printf("Place quickly found: %+v\n\n", a.PlaceID)
		return db.Instance.Save(a).Error == nil
	}
	// fmt.Printf("Location after: %+v\n\n", location)
	// Try a Nominatim request
	nominatim := getNominatimLocation(lat, long)
	if nominatim == nil {
		fmt.Printf("No location found for: %d, %f, %f\n\n", a.ID, lat, long)
		return false
	}
	// Create local DB record
	location.Display = nominatim.DisplayName
	location.Area = nominatim.GetArea()
	location.City = nominatim.GetCity()
	location.Country = nominatim.Address.Country
	location.CountryCode = nominatim.Address.CountryCode
	res := db.Instance.Create(&location)
	// fmt.Printf("Created location: %+v\n\n", location)
	if res.Error != nil {
		fmt.Printf("DB error: %+v\n", res.Error)
		return false
	}
	// Do we have a corresponding place already in our DB?
	a.PlaceID = location.GetPlaceID()
	if a.PlaceID == 0 {
		return false
	}
	return db.Instance.Save(a).Error == nil
}
