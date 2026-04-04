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
	err := func() error {
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

	if err != nil {
		var cachedLoc Location
		if loadErr := cache.Load("last_location.json", &cachedLoc); loadErr == nil {
			log.Printf("Using cached location due to error: %v (location: %s, %s)", err, cachedLoc.City, cachedLoc.Country)
			return &cachedLoc, nil
		}
		return nil, err
	}
	
	_ = cache.Save("last_location.json", loc)
	return &loc, nil
}
