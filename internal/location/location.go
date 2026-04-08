package location

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
)

type Location struct {
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone string  `json:"timezone"`
	Method   int     `json:"method"`
	IsManual bool    `json:"is_manual"`
}

type Service struct {
	apiURL string
}

func NewService() *Service {
	return &Service{
		apiURL: "https://ipwhois.app/json/",
	}
}

func (s *Service) GetLocation() (*Location, error) {
	var loc Location
	apiErr := func() error {
		resp, err := http.Get(s.apiURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("API status %d", resp.StatusCode)
		}
		
		return json.NewDecoder(resp.Body).Decode(&loc)
	}()

	// Load override if it exists
	var override Location
	if err := cache.Load("location_override.json", &override); err == nil {
		if override.City != "" {
			loc.City = override.City
			loc.IsManual = true
		}
		if override.Country != "" {
			loc.Country = override.Country
			loc.IsManual = true
		}
	}

	if apiErr != nil {
		// If API failed, but we have an override, we might still be okay if we have cached coordinates
		var cachedLoc Location
		if loadErr := cache.Load("last_location.json", &cachedLoc); loadErr == nil {
			// Merge override into cached location
			if loc.City == "" { loc.City = cachedLoc.City }
			if loc.Country == "" { loc.Country = cachedLoc.Country }
			loc.Lat = cachedLoc.Lat
			loc.Lon = cachedLoc.Lon
			loc.Timezone = cachedLoc.Timezone
			
			log.Printf("Using cached location with overrides due to API error: %v (location: %s, %s)", apiErr, loc.City, loc.Country)
			return &loc, nil
		}
		if loc.City != "" && loc.Country != "" {
			// We have an override but no full cache, return what we have (coordinates might be missing but timings API doesn't strictly need them if city/country provided)
			return &loc, nil
		}
		return nil, apiErr
	}
	
	_ = cache.Save("last_location.json", loc)
	return &loc, nil
}
