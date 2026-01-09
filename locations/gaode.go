package locations

import (
	"encoding/json"
	"io"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"math"
)

// WGS84(World) to GCJ02(China GPS) offline algo
func wgs84ToGCJ02(wgsLon, wgsLat float64) (float64, float64) {
	const pi = 3.1415926535897932384626
	const a = 6378245.0              
	const ee = 0.00669342162296594323 
	
	if outOfChina(wgsLon, wgsLat) {
		return wgsLon, wgsLat
	}
	
	dLat := transformLat(wgsLon-105.0, wgsLat-35.0)
	dLon := transformLon(wgsLon-105.0, wgsLat-35.0)
	
	radLat := wgsLat / 180.0 * pi
	magic := math.Sin(radLat)
	magic = 1 - ee*magic*magic
	sqrtMagic := math.Sqrt(magic)
	
	dLat = (dLat * 180.0) / ((a * (1 - ee)) / (magic * sqrtMagic) * pi)
	dLon = (dLon * 180.0) / (a / sqrtMagic * math.Cos(radLat) * pi)
	
	gcjLat := wgsLat + dLat
	gcjLon := wgsLon + dLon
	
	return gcjLon, gcjLat
}

func outOfChina(lon, lat float64) bool {
	return lon < 72.004 || lon > 137.8347 || lat < 0.8293 || lat > 55.8271
}

func transformLat(x, y float64) float64 {
	ret := -100.0 + 2.0*x + 3.0*y + 0.2*y*y + 0.1*x*y + 0.2*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(y*math.Pi) + 40.0*math.Sin(y/3.0*math.Pi)) * 2.0 / 3.0
	ret += (160.0*math.Sin(y/12.0*math.Pi) + 320*math.Sin(y*math.Pi/30.0)) * 2.0 / 3.0
	return ret
}

func transformLon(x, y float64) float64 {
	ret := 300.0 + x + 2.0*y + 0.1*x*x + 0.1*x*y + 0.1*math.Sqrt(math.Abs(x))
	ret += (20.0*math.Sin(6.0*x*math.Pi) + 20.0*math.Sin(2.0*x*math.Pi)) * 2.0 / 3.0
	ret += (20.0*math.Sin(x*math.Pi) + 40.0*math.Sin(x/3.0*math.Pi)) * 2.0 / 3.0
	ret += (150.0*math.Sin(x/12.0*math.Pi) + 300.0*math.Sin(x/30.0*math.Pi)) * 2.0 / 3.0
	return ret
}




func GetGaodeLocation(lat, long float64, apiKey string) *NominatimLocation {
	// Add throttling
	if time.Since(lastRequest) < throttling {
		time.Sleep(throttling - time.Since(lastRequest))
	}
	lastRequest = time.Now()
	gcj02Long, gcj02Lat := wgs84ToGCJ02(long, lat)
	url := fmt.Sprintf("https://restapi.amap.com/v3/geocode/regeo?key=%s&location=%f,%f&extensions=all&batch=false&roadlevel=0&output=JSON", apiKey, gcj02Long, gcj02Lat)
	log.Printf("Making request to: %s", url)
	
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Failed request to:", url, err)
		return nil
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response body:", err)
		return nil
	}

	// China has municipalities directly under the central government, and the Amap API does not display their city names.
	var gaodeResp struct {
		Status    string `json:"status"`
		Info      string `json:"info"`
		Regeocode struct {
			FormattedAddress string `json:"formatted_address"`
			AddressComponent struct {
				City        json.RawMessage `json:"city"`  // city : maybe string or empty array 
				Province    string          `json:"province"`
				District    string          `json:"district"`
				Township    string          `json:"township"`
				Country     string          `json:"country"`
				CountryCode string          `json:"country_code"`
				Citycode    string          `json:"citycode"`
			} `json:"addressComponent"`
		} `json:"regeocode"`
	}


	if err = json.Unmarshal(body, &gaodeResp); err != nil {
		log.Println("JSON decode error:", err)
		log.Println("URL:", url)
		return nil
	}


	if gaodeResp.Status != "1" {
		log.Printf("Gaode API error: Status=%s, Info=%s", gaodeResp.Status, gaodeResp.Info)
		return nil
	}

	// Get City Name (Processing Municipality Logic)
	var cityName string
	
	
	if len(gaodeResp.Regeocode.AddressComponent.City) > 0 && 
	   string(gaodeResp.Regeocode.AddressComponent.City) != "[]" {
		
		// Try to parse it as a string
		var cityStr string
		if err := json.Unmarshal(gaodeResp.Regeocode.AddressComponent.City, &cityStr); err == nil && cityStr != "" {
			cityName = cityStr
		} else {
			// Try to parse it as a string array
			var cityArr []string
			if err := json.Unmarshal(gaodeResp.Regeocode.AddressComponent.City, &cityArr); err == nil && len(cityArr) > 0 {
				cityName = cityArr[0]
			}
		}
	}

	// city field is empty
	if cityName == "" {
		// all Municipality cities : BEIJING SHANGHAI TIANJIN CHONGQING
		municipalities := map[string]string{
			"北京市": "北京",
			"上海市": "上海",
			"天津市": "天津",
			"重庆市": "重庆",
		}

		// Check for Direct-Controlled Municipality
		if city, ok := municipalities[gaodeResp.Regeocode.AddressComponent.Province]; ok {
			cityName = city
		} else if gaodeResp.Regeocode.AddressComponent.District != "" {
			cityName = gaodeResp.Regeocode.AddressComponent.District
		} else {
			cityName = gaodeResp.Regeocode.AddressComponent.Province
		}
	}

	result := &NominatimLocation{
		DisplayName: gaodeResp.Regeocode.FormattedAddress,
		Address: NominatimAddress{
			City:          cityName,
			Province:      gaodeResp.Regeocode.AddressComponent.Province,
			Neighbourhood: gaodeResp.Regeocode.AddressComponent.Township,
			Country:       gaodeResp.Regeocode.AddressComponent.Country,
			CountryCode:   gaodeResp.Regeocode.AddressComponent.CountryCode,
		},
	}

	if result.DisplayName == "" {
		parts := []string{}
		
		isMunicipality := false
		municipalityProvinces := []string{"北京市", "上海市", "天津市", "重庆市"}
		for _, mp := range municipalityProvinces {
			if result.Address.Province == mp {
				isMunicipality = true
				break
			}
		}

		if result.Address.Neighbourhood != "" {
			parts = append(parts, result.Address.Neighbourhood)
		}
		
		if gaodeResp.Regeocode.AddressComponent.District != "" && !isMunicipality {
			parts = append(parts, gaodeResp.Regeocode.AddressComponent.District)
		}
		
		if result.Address.City != "" {
			parts = append(parts, result.Address.City)
		}
		
		if result.Address.Province != "" && !isMunicipality {
			parts = append(parts, result.Address.Province)
		}
		
		if result.Address.Country != "" {
			parts = append(parts, result.Address.Country)
		}
		
		result.DisplayName = strings.Join(parts, ", ")
	}
	
	return result
}