package locations

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"server/models"
	"strings"
	"time"
)

var (
	client      = http.Client{}
	lastRequest = time.Now().Add(-10 * time.Second)
)

const (
	throttling = 3 * time.Second
)

type NominatimAddress struct {
	Aeroway       string `json:"aeroway"`
	Railway       string `json:"railway"`
	Place         string `json:"place"`
	Neighbourhood string `json:"neighbourhood"`
	City          string `json:"city"`
	Municipality  string `json:"municipality"`
	Province      string `json:"province"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
}

type NominatimLocation struct {
	DisplayName string           `json:"display_name"`
	Address     NominatimAddress `json:"address"`
}

func (n *NominatimLocation) GetCity() string {
	if n.Address.City != "" {
		return n.Address.City
	}
	if n.Address.Municipality != "" {
		return n.Address.Municipality
	}
	return n.Address.Province
}

func (n *NominatimLocation) GetArea() string {
	if n.Address.Aeroway != "" && len(n.Address.Aeroway) > 4 {
		if n.Address.Neighbourhood != "" {
			return n.Address.Aeroway + ", " + n.Address.Neighbourhood
		}
		return n.Address.Aeroway
	}
	if n.Address.Railway != "" {
		return n.Address.Railway
	}
	if n.Address.Place != "" {
		return n.Address.Place
	}
	if n.Address.Neighbourhood != "" {
		return n.Address.Neighbourhood
	}
	a := strings.Split(n.DisplayName, ",")
	city := n.GetCity()
	for i := len(a) - 1; i > 0; i-- {
		if strings.TrimLeft(a[i], " ") == city {
			return strings.TrimLeft(a[i-1], " ")
		}
	}
	if len(a) == 1 || len(a[0]) >= models.MinLocationDisplaySize {
		return a[0]
	}
	return a[0] + "," + a[1]
}

func getNominatimLocation(lat, long float64) *NominatimLocation {
	// Add throttling
	if time.Since(lastRequest) < throttling {
		time.Sleep(throttling - time.Since(lastRequest))
	}
	lastRequest = time.Now()

	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f", lat, long)
	log.Printf("Making request to: %s", url)
	req, _ := http.NewRequest("GET", url, nil)
	// TODO: not only English?
	req.Header.Set("accept-language", "en")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed request to:", url, err)
		return nil
	}
	result := &NominatimLocation{}
	decoder := json.NewDecoder(resp.Body)
	defer resp.Body.Close()

	if err = decoder.Decode(result); err != nil {
		log.Println(url, err)
		return nil
	}
	return result
}
