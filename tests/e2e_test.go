package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build binary
	tmpDir, err := os.MkdirTemp("", "salat-break-e2e-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	binaryPath = filepath.Join(tmpDir, "salat-break")
	cmd := exec.Command("go", "build", "-o", binaryPath, "../cmd/salat-break")
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to build binary: %v\nOutput: %s\n", err, string(out))
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestE2E_Notifications(t *testing.T) {
	// Setup cache override
	tmpCache, _ := os.MkdirTemp("", "salat-break-e2e-cache-*")
	defer os.RemoveAll(tmpCache)

	absMocks, _ := filepath.Abs("mocks")

	cmd := exec.Command(binaryPath, "--test-notify", "--notification-clear-delay", "100")
	cmd.Env = append(os.Environ(), 
		"PATH="+absMocks+":"+os.Getenv("PATH"),
		"HOME="+tmpCache,
	)
	
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	assert.NoError(t, err, out.String())

	// Verify notification log
	content, err := os.ReadFile("/tmp/salat-break-notifications.log")
	require.NoError(t, err)
	assert.Contains(t, string(content), "Title: Salat Break Test")
}

func TestE2E_MediaPause(t *testing.T) {
	// Setup cache override
	tmpCache, _ := os.MkdirTemp("", "salat-break-e2e-cache-*")
	defer os.RemoveAll(tmpCache)

	absMocks, _ := filepath.Abs("mocks")

	// Clear logs
	os.Remove("/tmp/salat-break-dbus.log")
	os.Remove("/tmp/salat-break-notifications.log")

	cmd := exec.Command(binaryPath, "--test-pause", "--notification-clear-delay", "100")
	cmd.Env = append(os.Environ(), 
		"PATH="+absMocks+":"+os.Getenv("PATH"),
		"HOME="+tmpCache,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	assert.NoError(t, err, out.String())

	// Verify DBUS log
	dbusLog, _ := os.ReadFile("/tmp/salat-break-dbus.log")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Spotify")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Chromium PWA")

	// Verify notification log
	notifyLog, _ := os.ReadFile("/tmp/salat-break-notifications.log")
	assert.Contains(t, string(notifyLog), "Title: Media Paused")
}

func TestE2E_LogicSimulation(t *testing.T) {
	cmd := exec.Command(binaryPath, "--test-logic")
	
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	assert.NoError(t, err, out.String())
	assert.Contains(t, out.String(), "--- Starting Logic Simulation ---")
	assert.Contains(t, out.String(), "Entering prayer window for Dhuhr (12:00)")
	assert.Contains(t, out.String(), "--- Simulation Complete ---")
}

func TestE2E_LocationDetection(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	// Setup cache override
	tmpCache, _ := os.MkdirTemp("", "salat-break-e2e-cache-*")
	defer os.RemoveAll(tmpCache)

	absMocks, _ := filepath.Abs("mocks")

	cmd := exec.Command(binaryPath, "--show-location")
	cmd.Env = append(os.Environ(), 
		"PATH="+absMocks+":"+os.Getenv("PATH"),
		"HOME="+tmpCache,
		"SALAT_BREAK_ALADHAN_URL="+mockServer.URL+"/v1",
		"SALAT_BREAK_BEACONDB_URL="+mockServer.URL+"/v1/geolocate",
		"SALAT_BREAK_IPINFO_URL="+mockServer.URL+"/json",
		"SALAT_BREAK_IPAPI_URL="+mockServer.URL+"/json",
		"SALAT_BREAK_IPWHOIS_URL="+mockServer.URL+"/json",
		"SALAT_BREAK_NOMINATIM_URL="+mockServer.URL+"/reverse",
		"SALAT_BREAK_IPIFY_URL="+mockServer.URL+"/api.ipify.org",
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	assert.NoError(t, err, out.String())
	
	// Should show city from mock server
	assert.Contains(t, out.String(), "City:      CI_City")
}
