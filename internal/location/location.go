package location

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
)

// Location represents a geographic position with metadata.
type Location struct {
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Accuracy float64 `json:"accuracy"` // Accuracy in meters
	Timezone string  `json:"timezone"`
	Method   int     `json:"method"`
	IsManual bool    `json:"is_manual"`
	Source   string  `json:"source"` // Which source provided this location
}

type Service struct {
	lastLoggedSource string
	Verbose          bool
}

func NewService() *Service {
	return &Service{}
}

// GetLocation resolves the user's location using a multi-source strategy,
// ordered by accuracy (best first):
//
//  1. Manual override (--lat/--lon or --city/--country)
//  2. GeoClue2 D-Bus (WiFi triangulation, GPS — the most accurate automated source)
//  3. WiFi AP scanning + geolocation API (Google/BeaconDB)
//  4. IP geolocation (consensus from multiple providers)
//  5. Cached location (last known good)
func (s *Service) GetLocation() (*Location, error) {
	// Load manual override if exists
	var override Location
	hasOverride := false
	if err := cache.Load("location_override.json", &override); err == nil {
		hasOverride = true
	}

	// If manual lat/lon is set, use it directly
	if hasOverride && override.Lat != 0 && override.Lon != 0 {
		override.IsManual = true
		override.Source = "manual-coords"
		s.logSourceOnce(override.Source, &override)

		// Enrich with reverse geocoding if city/country is empty
		if override.City == "" || override.Country == "" {
			s.enrichWithReverseGeocode(&override)
		}

		_ = cache.Save("last_location.json", override)
		return &override, nil
	}

	// Try automated sources in order of accuracy
	loc, err := s.resolveAutomated()

	if err != nil {
		// All automated sources failed — try cache
		var cachedLoc Location
		if loadErr := cache.Load("last_location.json", &cachedLoc); loadErr == nil {
			log.Printf("All location sources failed: %v. Using cached location: %s, %s (source: %s)",
				err, cachedLoc.City, cachedLoc.Country, cachedLoc.Source)
			cachedLoc.Source = "cached"

			// Apply manual city/country override on cached location
			if hasOverride {
				applyOverride(&cachedLoc, &override)
			}
			return &cachedLoc, nil
		}

		// If we have a manual city/country override but no coords, return that
		if hasOverride && override.City != "" {
			override.IsManual = true
			override.Source = "manual-city-only"
			return &override, nil
		}

		return nil, fmt.Errorf("all location sources failed and no cache available: %w", err)
	}

	// Apply manual overrides (city/country/method) on top of auto-detected coords
	if hasOverride {
		applyOverride(loc, &override)
	}

	// Reverse geocode if we have coords but no city (common for GeoClue2/WiFi)
	if loc.City == "" || loc.Country == "" {
		s.enrichWithReverseGeocode(loc)
	}

	s.logSourceOnce(loc.Source, loc)

	// Cache the resolved location
	_ = cache.Save("last_location.json", *loc)
	return loc, nil
}

// resolveAutomated tries each automated location source in order of accuracy.
func (s *Service) resolveAutomated() (*Location, error) {
	// 1. Try GeoClue2 D-Bus — most accurate (uses WiFi, GPS, cell towers)
	loc, err := tryGeoClue2()
	if err == nil && loc.Accuracy > 0 && loc.Accuracy < 1000 {
		return loc, nil
	}
	if err != nil {
		s.logVerbose("GeoClue2 unavailable: %v", err)
	}

	// 2. Try WiFi AP scanning + geolocation API
	wifiLoc, wifiErr := tryWiFiGeolocation()
	// If WiFi is highly accurate (< 5km), use it immediately
	if wifiErr == nil && wifiLoc.Accuracy > 0 && wifiLoc.Accuracy < 5000 {
		return wifiLoc, nil
	}
	if wifiErr != nil {
		s.logVerbose("WiFi geolocation failed/sparse: %v", wifiErr)
	}

	// 3. Try IP geolocation (Consensus)
	// We do this if GeoClue and WiFi failed or returned low accuracy
	ipLoc, ipErr := tryIPGeolocation()

	// 4. Decision Logic: Pick the best available result
	if wifiErr == nil && ipErr == nil {
		// If we have both, pick the one with better (lower) accuracy
		// Note: IP Accuracy is usually estimated at 50km for consensus
		if wifiLoc.Accuracy > 0 && wifiLoc.Accuracy < ipLoc.Accuracy {
			return wifiLoc, nil
		}
		return ipLoc, nil
	}

	if wifiErr == nil {
		return wifiLoc, nil
	}
	if ipErr == nil {
		return ipLoc, nil
	}

	return nil, fmt.Errorf("all automated sources failed: wifi=%v, ip=%v", wifiErr, ipErr)
}

func (s *Service) logVerbose(format string, v ...interface{}) {
	if s.Verbose {
		log.Printf(format, v...)
	}
}

// applyOverride merges manual overrides onto an auto-detected location.
func applyOverride(loc *Location, override *Location) {
	if override.City != "" {
		loc.City = override.City
		loc.IsManual = true
	}
	if override.Country != "" {
		loc.Country = override.Country
		loc.IsManual = true
	}
	if override.Method > 0 {
		loc.Method = override.Method
	}
	// Manual lat/lon override
	if override.Lat != 0 && override.Lon != 0 {
		loc.Lat = override.Lat
		loc.Lon = override.Lon
		loc.IsManual = true
	}
}

// enrichWithReverseGeocode uses a free reverse geocoding API to get city/country
// from coordinates. This is useful when GeoClue2 or WiFi gives us lat/lon only.
func (s *Service) enrichWithReverseGeocode(loc *Location) {
	if loc.Lat == 0 && loc.Lon == 0 {
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%.6f&lon=%.6f&zoom=10",
		loc.Lat, loc.Lon)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Reverse geocode: request creation failed: %v", err)
		return
	}
	req.Header.Set("User-Agent", "salat-break/1.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Reverse geocode: request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Reverse geocode: status %d", resp.StatusCode)
		return
	}

	var data struct {
		Address struct {
			City        string `json:"city"`
			Town        string `json:"town"`
			Village     string `json:"village"`
			Municipality string `json:"municipality"`
			County      string `json:"county"`
			Country     string `json:"country"`
			CountryCode string `json:"country_code"`
		} `json:"address"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Reverse geocode: decode failed: %v", err)
		return
	}

	// Pick the most specific place name available
	city := data.Address.City
	if city == "" {
		city = data.Address.Town
	}
	if city == "" {
		city = data.Address.Village
	}
	if city == "" {
		city = data.Address.Municipality
	}
	if city == "" {
		city = data.Address.County
	}

	if city != "" && loc.City == "" {
		loc.City = city
		log.Printf("Reverse geocode: resolved city to %s", city)
	}
	if data.Address.Country != "" && loc.Country == "" {
		loc.Country = data.Address.Country
		log.Printf("Reverse geocode: resolved country to %s", data.Address.Country)
	}
}

// logSourceOnce logs the location source only when it changes.
func (s *Service) logSourceOnce(source string, loc *Location) {
	if s.lastLoggedSource != source {
		log.Printf("Location resolved via [%s] (accuracy: %.0fm): lat=%.4f, lon=%.4f, city=%s, country=%s",
			source, loc.Accuracy, loc.Lat, loc.Lon, loc.City, loc.Country)
		s.lastLoggedSource = source
	}
}
