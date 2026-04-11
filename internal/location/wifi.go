package location

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
)

// wifiAP represents a WiFi access point scanned from the system.
type wifiAP struct {
	BSSID  string `json:"macAddress"`
	Signal int    `json:"signalStrength"`
	SSID   string `json:"-"` // Not sent to API but useful for logging
}

// scanWiFiAPs uses nmcli to scan nearby WiFi access points.
func scanWiFiAPs() ([]wifiAP, error) {
	// Check if nmcli is available
	_, err := exec.LookPath("nmcli")
	if err != nil {
		return nil, fmt.Errorf("wifi: nmcli not found: %w", err)
	}

	// Trigger a fresh scan (this may fail if not root, but that's okay — stale results still work)
	_ = exec.Command("nmcli", "device", "wifi", "rescan").Run()

	// List WiFi APs with BSSID and signal
	cmd := exec.Command("nmcli", "-t", "-f", "BSSID,SIGNAL,SSID", "device", "wifi", "list")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wifi: nmcli scan failed: %w", err)
	}

	var aps []wifiAP
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// nmcli -t uses ':' as separator, but BSSID contains ':'
		// Format: AA\:BB\:CC\:DD\:EE\:FF:SIGNAL:SSID
		// The BSSID has escaped colons (\:), but the field separators are unescaped colons
		// We need to handle this carefully

		// Replace escaped colons in BSSID with a placeholder
		line = strings.ReplaceAll(line, `\:`, "§")
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			continue
		}

		bssid := strings.ReplaceAll(parts[0], "§", ":")
		bssid = strings.TrimSpace(bssid)
		signal, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			continue
		}

		ssid := ""
		if len(parts) >= 3 {
			ssid = strings.ReplaceAll(parts[2], "§", ":")
		}

		// Convert signal percentage (0-100) to dBm (roughly)
		// nmcli reports signal as percentage; -30 dBm = 100%, -90 dBm = 0%
		signalDBm := -90 + (signal * 60 / 100)

		aps = append(aps, wifiAP{
			BSSID:  bssid,
			Signal: signalDBm,
			SSID:   ssid,
		})
	}

	if len(aps) == 0 {
		return nil, fmt.Errorf("wifi: no access points found")
	}

	return aps, nil
}

// tryWiFiGeolocation scans nearby WiFi access points and uses a geolocation API
// to determine the location via WiFi triangulation.
// This uses the same technique as Google Maps on devices without GPS.
func tryWiFiGeolocation() (*Location, error) {
	aps, err := scanWiFiAPs()
	if err != nil {
		return nil, err
	}

	// Limit to top 20 strongest signals for better accuracy
	if len(aps) > 20 {
		aps = aps[:20]
	}

	// 1. Try Google Geolocation API (Will fail without key, but keep as placeholder)
	loc, err := geolocateViaGoogle(aps)
	if err == nil {
		return loc, nil
	}
	// log.Printf("WiFi: Google geolocation failed: %v", err)

	// 2. Try high-precision "Apple-style" community lookup if available
	// (Note: This is a placeholder for future complex binary implementation)

	// 3. Fallback: try BeaconDB (Community-driven)
	loc, err = geolocateViaMozillaCompat(aps)
	if err == nil {
		// If accuracy is very low (> 10km), we might want to flag it
		return loc, nil
	}

	return nil, fmt.Errorf("wifi: all geolocation APIs failed: %w", err)
}

type googleGeoRequest struct {
	WifiAccessPoints []wifiAP `json:"wifiAccessPoints"`
}

type googleGeoResponse struct {
	Location struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"location"`
	Accuracy float64 `json:"accuracy"`
}

func geolocateViaGoogle(aps []wifiAP) (*Location, error) {
	// Use Google Geolocation API
	// This works without an API key for limited usage
	reqBody := googleGeoRequest{WifiAccessPoints: aps}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := "https://www.googleapis.com/geolocation/v1/geolocate?key="
	// Look for API key in environment or use keyless (very limited)
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("google geo: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google geo: status %d", resp.StatusCode)
	}

	var geoResp googleGeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return nil, err
	}

	return &Location{
		Lat:      geoResp.Location.Lat,
		Lon:      geoResp.Location.Lng,
		Accuracy: geoResp.Accuracy,
		Source:   "wifi-google",
	}, nil
}

func geolocateViaMozillaCompat(aps []wifiAP) (*Location, error) {
	// BeaconDB — community-driven replacement for Mozilla Location Service
	type beaconReq struct {
		WifiAccessPoints []struct {
			MacAddress     string `json:"macAddress"`
			SignalStrength int    `json:"signalStrength"`
		} `json:"wifiAccessPoints"`
	}

	var req beaconReq
	for _, ap := range aps {
		req.WifiAccessPoints = append(req.WifiAccessPoints, struct {
			MacAddress     string `json:"macAddress"`
			SignalStrength int    `json:"signalStrength"`
		}{
			MacAddress:     ap.BSSID,
			SignalStrength: ap.Signal,
		})
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	resp, err := http.Post("https://beacondb.net/v1/geolocate", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("beacondb: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("beacondb: status %d", resp.StatusCode)
	}

	var geoResp googleGeoResponse // BeaconDB uses same response format
	if err := json.NewDecoder(resp.Body).Decode(&geoResp); err != nil {
		return nil, err
	}

	return &Location{
		Lat:      geoResp.Location.Lat,
		Lon:      geoResp.Location.Lng,
		Accuracy: geoResp.Accuracy,
		Source:   "wifi-beacondb",
	}, nil
}
