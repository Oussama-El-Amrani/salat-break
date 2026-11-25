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

	// Verify DBUS log: Spotify, VLC, and Chromium PWA are paused; Firefox tutorial is NOT paused
	dbusLog, _ := os.ReadFile("/tmp/salat-break-dbus.log")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Spotify")
	assert.Contains(t, string(dbusLog), "Pause signal sent to VLC")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Chromium PWA")
	assert.NotContains(t, string(dbusLog), "Pause signal sent to Firefox")

	// Verify notification log
	notifyLog, _ := os.ReadFile("/tmp/salat-break-notifications.log")
	assert.Contains(t, string(notifyLog), "Title: Media Paused")
}

func TestE2E_ToggleProviderAndAllowInSalatSlot(t *testing.T) {
	tmpCache, _ := os.MkdirTemp("", "salat-break-e2e-cache-*")
	defer os.RemoveAll(tmpCache)

	absMocks, _ := filepath.Abs("mocks")
	env := append(os.Environ(),
		"PATH="+absMocks+":"+os.Getenv("PATH"),
		"HOME="+tmpCache,
	)

	// 1. Toggle VLC ON (allow in Salat slot)
	var toggleOnOut bytes.Buffer
	cmdToggleOn := exec.Command(binaryPath, "--toggle-provider", "vlc")
	cmdToggleOn.Env = env
	cmdToggleOn.Stdout = &toggleOnOut
	cmdToggleOn.Stderr = &toggleOnOut
	require.NoError(t, cmdToggleOn.Run(), toggleOnOut.String())
	assert.Contains(t, toggleOnOut.String(), `Provider "vlc" is now ALLOWED`)

	// 2. Verify --list-providers shows VLC as allowed and Spotify as paused
	var listOut bytes.Buffer
	cmdList := exec.Command(binaryPath, "--list-providers")
	cmdList.Env = env
	cmdList.Stdout = &listOut
	cmdList.Stderr = &listOut
	require.NoError(t, cmdList.Run(), listOut.String())
	assert.Contains(t, listOut.String(), "vlc")
	assert.Contains(t, listOut.String(), "Allowed during Salat")
	assert.Contains(t, listOut.String(), "Allowed Providers in Salat Slot: vlc")

	// 3. Run --test-pause: Spotify and Chromium PWA must be paused, VLC must NOT be paused
	os.Remove("/tmp/salat-break-dbus.log")
	var pauseOut bytes.Buffer
	cmdPause := exec.Command(binaryPath, "--test-pause", "--notification-clear-delay", "100")
	cmdPause.Env = env
	cmdPause.Stdout = &pauseOut
	cmdPause.Stderr = &pauseOut
	require.NoError(t, cmdPause.Run(), pauseOut.String())

	dbusLog, _ := os.ReadFile("/tmp/salat-break-dbus.log")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Spotify")
	assert.Contains(t, string(dbusLog), "Pause signal sent to Chromium PWA")
	assert.NotContains(t, string(dbusLog), "Pause signal sent to VLC")

	// 4. Toggle VLC OFF again (disallow in Salat slot)
	var toggleOffOut bytes.Buffer
	cmdToggleOff := exec.Command(binaryPath, "--toggle-provider", "vlc")
	cmdToggleOff.Env = env
	cmdToggleOff.Stdout = &toggleOffOut
	cmdToggleOff.Stderr = &toggleOffOut
	require.NoError(t, cmdToggleOff.Run(), toggleOffOut.String())
	assert.Contains(t, toggleOffOut.String(), `Provider "vlc" is now DISALLOWED`)

	// 5. Run --test-pause again: VLC must now be paused
	os.Remove("/tmp/salat-break-dbus.log")
	var pauseAgainOut bytes.Buffer
	cmdPauseAgain := exec.Command(binaryPath, "--test-pause", "--notification-clear-delay", "100")
	cmdPauseAgain.Env = env
	cmdPauseAgain.Stdout = &pauseAgainOut
	cmdPauseAgain.Stderr = &pauseAgainOut
	require.NoError(t, cmdPauseAgain.Run(), pauseAgainOut.String())

	dbusLogAfter, _ := os.ReadFile("/tmp/salat-break-dbus.log")
	assert.Contains(t, string(dbusLogAfter), "Pause signal sent to VLC")
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

func TestE2E_ShowTimings(t *testing.T) {
	mockServer := NewMockAPIServer()
	defer mockServer.Close()

	// Setup cache override
	tmpCache, _ := os.MkdirTemp("", "salat-break-e2e-cache-*")
	defer os.RemoveAll(tmpCache)

	absMocks, _ := filepath.Abs("mocks")

	cmd := exec.Command(binaryPath, "--show-timings")
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

	outputStr := out.String()
	assert.Contains(t, outputStr, "SALAT BREAK TIMINGS")
	assert.Contains(t, outputStr, "CI_City")
	assert.Contains(t, outputStr, "CI_Country")
	assert.Contains(t, outputStr, "Fajr")
	assert.Contains(t, outputStr, "Dhuhr")
	assert.Contains(t, outputStr, "Asr")
	assert.Contains(t, outputStr, "Maghrib")
	assert.Contains(t, outputStr, "Isha")
}
