package models

import (
	"server/db"
	"strings"
)

const (
	MinLocationDisplaySize = 5
)

// Location is used as cache to avoid hammering the Geocoding service
type Location struct {
	GpsLat      float64 `gorm:"type:double;primaryKey"` // Rounded to 0.0001
	GpsLong     float64 `gorm:"type:double;primaryKey"` // Rounded to 0.0001
	Display     string  `gorm:"type:varchar(250)"`
	Area        string  `gorm:"type:varchar(100)"`
	City        string  `gorm:"type:varchar(100)"`
	Country     string  `gorm:"type:varchar(100)"`
	CountryCode string  `gorm:"type:varchar(10)"`
}

func (n *Location) GetShortDisplay() string {
	r := strings.SplitN(n.Display, ",", 3)
	if len(r) == 1 || len(r[0]) >= MinLocationDisplaySize {
		return r[0]
	}
	return r[0] + "," + r[1]
}

func (location *Location) GetPlaceID() uint64 {
	if location.Area == "" && location.City == "" && location.Country == "" {
		return 0
	}
	place := Place{
		Area:    location.Area,
		City:    location.City,
		Country: location.Country,
	}
	if (location.Area != "" || location.City != "") && location.Country != "" {
		db.Instance.Debug().Where(&place).FirstOrCreate(&place)
	} else {
		db.Instance.Debug().Where(&place, "area", "city", "country").Limit(1).Find(&place)
	}
	// fmt.Printf("The place is: %+v\n", place)
	return place.ID
}
