package locations

import (
	"log"
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
	for {
		asset := getNextForProcessing(lastProcessedID)
		if asset.ID == 0 {
			// Nothing to process...
			time.Sleep(60 * time.Second)
			lastProcessedID = 0
			continue
		}
		_ = process(&asset)
		lastProcessedID = asset.ID
	}
}

func process(a *models.Asset) bool {
	// Try first local DB
	location := a.GetRoughLocation()
	var locations []models.Location
	db.Instance.Where("gps_lat=? and gps_long=?", location.GpsLat, location.GpsLong).Find(&locations)
	if len(locations) > 0 {
		placeID := locations[0].GetPlaceID()
		if placeID > 0 {
			a.PlaceID = &placeID
			return db.Instance.Save(a).Error == nil
		}
	}
	// Try a Nominatim request
	nominatim := getNominatimLocation(location.GpsLat, location.GpsLong)
	if nominatim == nil {
		log.Printf("No location found for: %d, %f, %f", a.ID, location.GpsLat, location.GpsLong)
		return false
	}
	// Create local DB record
	location.Display = nominatim.DisplayName
	location.Area = nominatim.GetArea()
	location.City = nominatim.GetCity()
	location.Country = nominatim.Address.Country
	location.CountryCode = nominatim.Address.CountryCode
	res := db.Instance.Create(&location)
	if res.Error != nil {
		log.Printf("DB error: %+v", res.Error)
		return false
	}
	// Do we have a corresponding place already in our DB?
	placeID := location.GetPlaceID()
	if placeID == 0 {
		return false
	}
	a.PlaceID = &placeID
	return db.Instance.Save(a).Error == nil
}
