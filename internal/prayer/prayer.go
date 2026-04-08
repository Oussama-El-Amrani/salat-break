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
	apiURL             string
	lastLoggedCacheKey string
	Method             int
}

func NewService() *Service {
	return &Service{
		apiURL: "https://api.aladhan.com/v1/timingsByCity/",
	}
}

func (s *Service) GetPrayerTimes(loc *location.Location) (*PrayerTimes, error) {
	date := time.Now().Format("02-01-2006")
	safeCity := cache.SanitizeName(loc.City)
	safeCountry := cache.SanitizeName(loc.Country)
	cacheKey := fmt.Sprintf("prayer_times_%s_%s_%s_m%d.json", strings.ToLower(safeCity), strings.ToLower(safeCountry), date, s.Method)
	
	var cachedPT PrayerTimes
	if err := cache.Load(cacheKey, &cachedPT); err == nil {
		if s.lastLoggedCacheKey != cacheKey {
			modTime, _ := cache.GetModTime(cacheKey)
			log.Printf("Using cached prayer times for %s, %s (Date: %s, Cached at: %s)", 
				loc.City, loc.Country, date, modTime.Format("2006-01-02 15:04:05"))
			log.Printf("Today's Timings: %s", cachedPT.FormatTimings())
			s.lastLoggedCacheKey = cacheKey
		}
		return &cachedPT, nil
	}

	log.Printf("Fetching fresh prayer times from API for %s, %s (Date: %s, Method: %d)...", loc.City, loc.Country, date, s.Method)
	
	params := url.Values{}
	params.Add("city", loc.City)
	params.Add("country", loc.Country)
	if s.Method > 0 {
		params.Add("method", fmt.Sprintf("%d", s.Method))
	}

	url := fmt.Sprintf("%s%s?%s", s.apiURL, date, params.Encode())
	resp, err := http.Get(url)
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
	log.Printf("Successfully fetched and cached prayer times for %s.", loc.City)
	log.Printf("Today's Timings: %s", pt.FormatTimings())
	return &pt, nil
}
