package prayer

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
	"github.com/Oussama-El-Amrani/salat-break/internal/location"
)

type PrayerTimes struct {
	Data struct {
		Timings map[string]string `json:"timings"`
	} `json:"data"`
}

func (pt *PrayerTimes) FormatTimings() string {
	prayers := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	var results []string
	for _, name := range prayers {
		if t, ok := pt.Data.Timings[name]; ok {
			results = append(results, fmt.Sprintf("%s: %s", name, t))
		}
	}
	return strings.Join(results, " | ")
}

type Service struct {
	lastLoggedCacheKey string
	Method             int
}

func NewService() *Service {
	return &Service{}
}

// GetPrayerTimes fetches prayer times for the given location.
// It uses lat/lon coordinates when available (more accurate) and falls back
// to city/country name when coordinates are not available.
func (s *Service) GetPrayerTimes(loc *location.Location) (*PrayerTimes, error) {
	date := time.Now().Format("02-01-2006")

	// Build a unique cache key based on the actual parameters used
	cacheKey := s.buildCacheKey(loc, date)

	var cachedPT PrayerTimes
	if err := cache.Load(cacheKey, &cachedPT); err == nil {
		if s.lastLoggedCacheKey != cacheKey {
			modTime, _ := cache.GetModTime(cacheKey)
			log.Printf("Using cached prayer times (Date: %s, Cached at: %s, Source: %s)",
				date, modTime.Format("2006-01-02 15:04:05"), loc.Source)
			log.Printf("Location: lat=%.4f, lon=%.4f (%s, %s)",
				loc.Lat, loc.Lon, loc.City, loc.Country)
			log.Printf("Today's Timings: %s", cachedPT.FormatTimings())
			s.lastLoggedCacheKey = cacheKey
		}
		return &cachedPT, nil
	}

	// Determine which API endpoint to use
	var apiURL string
	if loc.Lat != 0 || loc.Lon != 0 {
		// Use coordinate-based API (more accurate)
		apiURL = s.buildCoordinateURL(loc, date)
		log.Printf("Fetching prayer times by coordinates: lat=%.4f, lon=%.4f (source: %s)",
			loc.Lat, loc.Lon, loc.Source)
	} else {
		// Fall back to city-based API
		apiURL = s.buildCityURL(loc, date)
		log.Printf("Fetching prayer times by city: %s, %s (source: %s)",
			loc.City, loc.Country, loc.Source)
		if !loc.IsManual {
			log.Printf("Tip: For better accuracy, set your coordinates: salat-break --lat 33.5731 --lon -7.5898")
		}
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var pt PrayerTimes
	if err := json.NewDecoder(resp.Body).Decode(&pt); err != nil {
		return nil, err
	}

	_ = cache.Save(cacheKey, pt)
	s.lastLoggedCacheKey = cacheKey
	log.Printf("Successfully fetched and cached prayer times.")
	log.Printf("Today's Timings: %s", pt.FormatTimings())
	return &pt, nil
}

// buildCoordinateURL builds the Aladhan API URL using lat/lon coordinates.
// This is more accurate than city-based lookup because it doesn't depend on
// the API's geocoding of city names (which can be ambiguous).
func (s *Service) buildCoordinateURL(loc *location.Location, date string) string {
	params := url.Values{}
	params.Add("latitude", fmt.Sprintf("%.6f", loc.Lat))
	params.Add("longitude", fmt.Sprintf("%.6f", loc.Lon))
	if s.Method > 0 {
		params.Add("method", fmt.Sprintf("%d", s.Method))
	}
	if loc.Timezone != "" {
		params.Add("timezonestring", loc.Timezone)
	}
	return fmt.Sprintf("https://api.aladhan.com/v1/timings/%s?%s", date, params.Encode())
}

// buildCityURL builds the Aladhan API URL using city/country names (fallback).
func (s *Service) buildCityURL(loc *location.Location, date string) string {
	params := url.Values{}
	params.Add("city", loc.City)
	params.Add("country", loc.Country)
	if s.Method > 0 {
		params.Add("method", fmt.Sprintf("%d", s.Method))
	}
	return fmt.Sprintf("https://api.aladhan.com/v1/timingsByCity/%s?%s", date, params.Encode())
}

// buildCacheKey creates a unique cache key based on the location parameters used.
func (s *Service) buildCacheKey(loc *location.Location, date string) string {
	if loc.Lat != 0 || loc.Lon != 0 {
		// Round coordinates to 2 decimal places for cache key stability
		// (minor GPS drift shouldn't cause cache misses — 0.01° ≈ 1.1km)
		return fmt.Sprintf("prayer_times_%.2f_%.2f_%s_m%d.json",
			loc.Lat, loc.Lon, date, s.Method)
	}
	safeCity := cache.SanitizeName(loc.City)
	safeCountry := cache.SanitizeName(loc.Country)
	return fmt.Sprintf("prayer_times_%s_%s_%s_m%d.json",
		strings.ToLower(safeCity), strings.ToLower(safeCountry), date, s.Method)
}
