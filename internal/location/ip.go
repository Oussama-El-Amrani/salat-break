package location

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
)

// tryIPGeolocation queries multiple IP geolocation providers in parallel
// and uses a consensus (median) of coordinates for better accuracy.
// This is the least accurate method but the most reliable fallback.
func tryIPGeolocation(client HTTPClient) (*Location, error) {
	type result struct {
		loc *Location
		err error
	}

	providers := []struct {
		name string
		fn   func(HTTPClient) (*Location, error)
	}{
		{"ipinfo.io", fetchFromIPInfo},
		{"ip-api.com", fetchFromIPAPI},
		{"ipwhois.app", fetchFromIPWhois},
	}

	results := make(chan result, len(providers))
	var wg sync.WaitGroup

	for _, p := range providers {
		wg.Add(1)
		go func(name string, fn func(HTTPClient) (*Location, error)) {
			defer wg.Done()
			loc, err := fn(client)
			if err != nil {
				logVerbose("IP geolocation [%s]: failed: %v", name, err)
			} else {
				logVerbose("IP geolocation [%s]: lat=%.4f, lon=%.4f, city=%s", name, loc.Lat, loc.Lon, loc.City)
			}
			results <- result{loc, err}
		}(p.name, p.fn)
	}

	// Close channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect successful results
	var locs []*Location
	for r := range results {
		if r.err == nil && r.loc != nil {
			locs = append(locs, r.loc)
		}
	}

	if len(locs) == 0 {
		return nil, fmt.Errorf("ip geolocation: all providers failed")
	}

	// Use median of coordinates for consensus
	consensus := medianLocation(locs)
	consensus.Source = "ip-consensus"
	consensus.Accuracy = 25000 // 25km estimate for consensus

	// If only one provider succeeded, use its data directly
	if len(locs) == 1 {
		locs[0].Source = "ip-single"
		return locs[0], nil
	}

	// Use city/country from the closest provider to median
	closest := findClosest(consensus, locs)
	consensus.City = closest.City
	consensus.Country = closest.Country
	consensus.Timezone = closest.Timezone
	consensus.CountryCode = closest.CountryCode

	logVerbose("IP geolocation consensus (%d providers): lat=%.4f, lon=%.4f, city=%s, country_code=%s",
		len(locs), consensus.Lat, consensus.Lon, consensus.City, consensus.CountryCode)

	return consensus, nil
}

func fetchFromIPInfo(client HTTPClient) (*Location, error) {
	url := getEnvURL("SALAT_BREAK_IPINFO_URL", "https://ipinfo.io/json")
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var data struct {
		City     string `json:"city"`
		Country  string `json:"country"`
		Loc      string `json:"loc"` // "lat,lon"
		Timezone string `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var lat, lon float64
	_, err = fmt.Sscanf(data.Loc, "%f,%f", &lat, &lon)
	if err != nil {
		return nil, fmt.Errorf("cannot parse loc %q: %w", data.Loc, err)
	}

	return &Location{
		City:     data.City,
		CountryCode: data.Country, // ipinfo returns 2-letter code in 'country'
		Lat:         lat,
		Lon:         lon,
		Accuracy:    50000, // 50km estimate
		Timezone:    data.Timezone,
	}, nil
}

func fetchFromIPAPI(client HTTPClient) (*Location, error) {
	url := getEnvURL("SALAT_BREAK_IPAPI_URL", "http://ip-api.com/json/?fields=city,country,lat,lon,timezone")
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var data struct {
		City        string  `json:"city"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &Location{
		City:     data.City,
		Country:     data.Country,
		CountryCode: data.CountryCode,
		Lat:      data.Lat,
		Lon:      data.Lon,
		Accuracy: 50000,
		Timezone: data.Timezone,
	}, nil
}


func fetchFromIPWhois(client HTTPClient) (*Location, error) {
	url := getEnvURL("SALAT_BREAK_IPWHOIS_URL", "https://ipwhois.app/json/")
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	var data struct {
		City        string  `json:"city"`
		Country     string  `json:"country"`
		CountryCode string  `json:"country_code"`
		Lat         float64 `json:"latitude"`
		Lon         float64 `json:"longitude"`
		Timezone    string  `json:"timezone"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &Location{
		City:     data.City,
		Country:     data.Country,
		CountryCode: data.CountryCode,
		Lat:      data.Lat,
		Lon:      data.Lon,
		Accuracy: 50000,
		Timezone: data.Timezone,
	}, nil
}

// medianLocation computes the median latitude and longitude from a list of locations.
// Median is more robust than mean against outliers (e.g., one provider being wildly wrong).
func medianLocation(locs []*Location) *Location {
	lats := make([]float64, len(locs))
	lons := make([]float64, len(locs))
	for i, l := range locs {
		lats[i] = l.Lat
		lons[i] = l.Lon
	}
	sort.Float64s(lats)
	sort.Float64s(lons)

	mid := len(lats) / 2
	return &Location{
		Lat: lats[mid],
		Lon: lons[mid],
	}
}

// findClosest finds the location from the list that is geographically closest
// to the target location (for inheriting city/country metadata).
func findClosest(target *Location, locs []*Location) *Location {
	if len(locs) == 0 {
		return target
	}
	closest := locs[0]
	minDist := haversineDistance(target.Lat, target.Lon, closest.Lat, closest.Lon)
	for _, l := range locs[1:] {
		d := haversineDistance(target.Lat, target.Lon, l.Lat, l.Lon)
		if d < minDist {
			minDist = d
			closest = l
		}
	}
	return closest
}

// haversineDistance calculates the distance between two lat/lon points in km.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in km
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
