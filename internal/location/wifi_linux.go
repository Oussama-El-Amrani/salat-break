//go:build linux

package location

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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
