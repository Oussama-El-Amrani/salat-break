package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
)

func NewMockAPIServer() *httptest.Server {
	mux := http.NewServeMux()

	// Mock Aladhan Timings
	mux.HandleFunc("/v1/timings/", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": map[string]interface{}{
				"timings": map[string]string{
					"Fajr":    "05:00",
					"Dhuhr":   "12:00",
					"Asr":     "15:30",
					"Maghrib": "18:30",
					"Isha":    "20:00",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Mock Aladhan Calendar
	mux.HandleFunc("/v1/calendar", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{
					"timings": map[string]string{"Fajr": "05:00", "Dhuhr": "12:00", "Asr": "15:30", "Maghrib": "18:30", "Isha": "20:00"},
					"date": map[string]interface{}{
						"gregorian": map[string]string{"date": "01-01-2026"},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Mock IPify
	mux.HandleFunc("/api.ipify.org", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("1.2.3.4"))
	})

	// Mock IPInfo
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"city":     "CI_City",
			"country":  "CI",
			"loc":      "34.0, -5.0",
			"timezone": "Africa/Casablanca",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Mock BeaconDB
	mux.HandleFunc("/v1/geolocate", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"location": map[string]float64{"lat": 34.0, "lng": -5.0},
			"accuracy": 100.0,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	// Mock Nominatim (Reverse Geocode)
	mux.HandleFunc("/reverse", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"address": map[string]string{
				"city":         "CI_City",
				"country":      "CI_Country",
				"country_code": "CI",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	return httptest.NewServer(mux)
}
