package processing

import (
	"log"
	"server/config"
	"server/db"
	"server/locations"
	"server/models"
	"server/storage"
)

type location struct{}

func (l *location) shouldHandle(asset *models.Asset) bool {
	return asset.GpsLat != nil && asset.GpsLong != nil && asset.PlaceID == nil
}

func (l *location) requiresContent(asset *models.Asset) bool {
	return false
}

func (l *location) process(asset *models.Asset, storage storage.StorageAPI) (int, func()) {
	// Try first local DB
	location := asset.GetRoughLocation()
	var result []models.Location
	db.Instance.Where("gps_lat=? and gps_long=?", location.GpsLat, location.GpsLong).Find(&result)
	if len(result) > 0 {
		placeID := result[0].GetPlaceID()
		if placeID > 0 {
			asset.PlaceID = &placeID
			if db.Instance.Save(asset).Error != nil {
				return FailedDB, nil
			}
			return Done, nil
		}
	}
	var nominatim *locations.NominatimLocation
	if config.GAODE_API_KEY != "" {
		// Try Gaode Maps API
		nominatim = locations.GetGaodeLocation(location.GpsLat, location.GpsLong, config.GAODE_API_KEY)
	} else {
	// Try a Nominatim request
		nominatim = locations.GetNominatimLocation(location.GpsLat, location.GpsLong)
	}
	if nominatim == nil {
		log.Printf("No location found for: %d, %f, %f", asset.ID, location.GpsLat, location.GpsLong)
		return Failed, nil
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
		return FailedDB, nil
	}
	// Do we have a corresponding place already in our DB?
	placeID := location.GetPlaceID()
	if placeID == 0 {
		return Failed, nil
	}
	asset.PlaceID = &placeID
	if db.Instance.Save(asset).Error != nil {
		return FailedDB, nil
	}
	return Done, nil
}
