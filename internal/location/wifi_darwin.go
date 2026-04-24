//go:build darwin

package location

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// scanWiFiAPs uses the macOS airport utility to scan nearby WiFi access points.
func scanWiFiAPs() ([]wifiAP, error) {
	// The airport utility is in a deep path
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	
	cmd := exec.Command(airportPath, "-s")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wifi: airport scan failed: %w", err)
	}

	var aps []wifiAP
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, fmt.Errorf("wifi: no access points found in airport output")
	}

	// Skip the header
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// airport -s output is fixed-width-ish, but splitting by space works if we are careful
		// SSID can contain spaces, but BSSID is always a MAC address
		// RSSI is always a negative number
		
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Find BSSID (MAC address format)
		var bssid string
		var rssi int
		var bssidIdx int
		
		for i, field := range fields {
			if strings.Contains(field, ":") && len(field) == 17 {
				bssid = field
				bssidIdx = i
				break
			}
		}

		if bssid == "" {
			continue
		}

		// RSSI is usually the field after BSSID or before it depending on SSID spaces
		// Actually RSSI is usually right after BSSID in Fields if SSID had no spaces
		// If SSID has spaces, Fields[0...N] is SSID.
		// Let's assume RSSI is the field following BSSID or preceding it.
		// In `airport -s` output: SSID BSSID RSSI ...
		
		if bssidIdx+1 < len(fields) {
			val, err := strconv.Atoi(fields[bssidIdx+1])
			if err == nil {
				rssi = val
			}
		}

		if rssi == 0 && bssidIdx > 0 {
			val, err := strconv.Atoi(fields[bssidIdx-1])
			if err == nil {
				rssi = val
			}
		}

		aps = append(aps, wifiAP{
			BSSID:  bssid,
			Signal: rssi,
		})
	}

	if len(aps) == 0 {
		return nil, fmt.Errorf("wifi: no access points found")
	}

	return aps, nil
}
