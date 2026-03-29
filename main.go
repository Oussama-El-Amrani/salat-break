package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

func sanitizeName(name string) string {
	// Remove anything that isn't alphanumeric or safe characters to prevent path traversal
	reg := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	return reg.ReplaceAllString(name, "_")
}

func getCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/salat-break"
	}
	dir := filepath.Join(home, ".cache", "salat-break")
	_ = os.MkdirAll(dir, 0755)
	return dir
}

func saveCache(name string, data interface{}) error {
	path := filepath.Join(getCacheDir(), name)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewEncoder(file).Encode(data)
}

func loadCache(name string, target interface{}) error {
	path := filepath.Join(getCacheDir(), name)
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(target)
}

type Location struct {
	City     string  `json:"city"`
	Country  string  `json:"country"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Timezone string  `json:"timezone"`
}

type PrayerTimes struct {
	Data struct {
		Timings map[string]string `json:"timings"`
	} `json:"data"`
}

var notificationsSent = make(map[string]time.Time)
var lastLoggedCacheKey string
var inPrayerBreak bool
var lastHandledPrayer string
var playersToResume []string

func getCacheModTime(name string) time.Time {
	path := filepath.Join(getCacheDir(), name)
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

func sendNotification(title, message string) {
	log.Printf("Sending notification: %s - %s", title, message)
	// Use notify-send for desktop notifications
	cmd := exec.Command("notify-send", "-i", "appointment-soon", title, message)
	err := cmd.Run()
	if err != nil {
		log.Printf("Error sending notification: %v", err)
	}
}

func getBrowserLocation() (*Location, error) {
	var loc Location
	err := func() error {
		// Using ipwhois.app which provides free HTTPS and compatible field names
		resp, err := http.Get("https://ipwhois.app/json/")
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
		if loadErr := loadCache("last_location.json", &cachedLoc); loadErr == nil {
			log.Printf("Using cached location due to error: %v (location: %s, %s)", err, cachedLoc.City, cachedLoc.Country)
			return &cachedLoc, nil
		}
		return nil, err
	}
	
	_ = saveCache("last_location.json", loc)
	return &loc, nil
}

func getPrayerTimes(loc *Location) (*PrayerTimes, error) {
	date := time.Now().Format("02-01-2006")
	safeCity := sanitizeName(loc.City)
	safeCountry := sanitizeName(loc.Country)
	cacheKey := fmt.Sprintf("prayer_times_%s_%s_%s.json", strings.ToLower(safeCity), strings.ToLower(safeCountry), date)
	
	var cachedPT PrayerTimes
	if err := loadCache(cacheKey, &cachedPT); err == nil {
		if lastLoggedCacheKey != cacheKey {
			modTime := getCacheModTime(cacheKey)
			log.Printf("Using cached prayer times for %s, %s (Date: %s, Cached at: %s)", 
				loc.City, loc.Country, date, modTime.Format("2006-01-02 15:04:05"))
			lastLoggedCacheKey = cacheKey
		}
		return &cachedPT, nil
	}

	log.Printf("Fetching fresh prayer times from API for %s, %s (Date: %s)...", loc.City, loc.Country, date)
	url := fmt.Sprintf("https://api.aladhan.com/v1/timingsByCity/%s?city=%s&country=%s&method=2", date, loc.City, loc.Country)
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
	
	_ = saveCache(cacheKey, pt)
	lastLoggedCacheKey = cacheKey
	log.Printf("Successfully fetched and cached prayer times for %s.", loc.City)
	return &pt, nil
}

func getAllPlayers() []string {
	cmd := exec.Command("dbus-send", "--session", "--dest=org.freedesktop.DBus", "--type=method_call", "--print-reply", "/org/freedesktop/DBus", "org.freedesktop.DBus.ListNames")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error listing DBus names: %v", err)
		return nil
	}

	// Simple parsing of dbus-send output
	var players []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "org.mpris.MediaPlayer2.") {
			// Extract service name and VALIDATE it
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				playerName := parts[1]
				if strings.HasPrefix(playerName, "org.mpris.MediaPlayer2.") {
					players = append(players, playerName)
				}
			}
		}
	}
	return players
}

func getMetadata(player string) map[string]string {
	metadata := make(map[string]string)
	cmd := exec.Command("dbus-send", "--print-reply", "--session", "--dest="+player, "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Properties.Get", "string:org.mpris.MediaPlayer2.Player", "string:Metadata")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return metadata
	}

	// Very simple parsing of the DBus reply
	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "string \"xesam:title\"") || strings.Contains(line, "string \"xesam:artist\"") {
			// The next line usually contains the value
			if i+1 < len(lines) {
				valLine := strings.TrimSpace(lines[i+1])
				parts := strings.Split(valLine, "\"")
				if len(parts) >= 2 {
					if strings.Contains(line, "title") {
						metadata["title"] = parts[1]
					} else {
						metadata["artist"] = parts[1]
					}
				}
			}
		}
	}
	return metadata
}

func isMusic(player string, title, artist string) bool {
	player = strings.ToLower(player)
	title = strings.ToLower(title)
	artist = strings.ToLower(artist)

	// 1. Player identity
	musicPlayers := []string{"spotify", "youtube_music", "rhythmbox", "clementine", "mpd", "audacious", "music"}
	for _, mp := range musicPlayers {
		if strings.Contains(player, mp) {
			return true
		}
	}

	// 2. Title keywords
	musicKeywords := []string{"music", "song", "official video", "official audio", "lyrics", "cover", "remix", "album", "playlist", "feat."}
	for _, kw := range musicKeywords {
		if strings.Contains(title, kw) {
			return true
		}
	}

	return false
}

func getPlaybackStatus(player string) string {
	cmd := exec.Command("dbus-send", "--print-reply", "--session", "--dest="+player, "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Properties.Get", "string:org.mpris.MediaPlayer2.Player", "string:PlaybackStatus")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	outStr := string(output)
	if strings.Contains(outStr, "\"Playing\"") {
		return "Playing"
	}
	if strings.Contains(outStr, "\"Paused\"") {
		return "Paused"
	}
	if strings.Contains(outStr, "\"Stopped\"") {
		return "Stopped"
	}
	return ""
}

func pauseAllPlayers() {
	playersToResume = []string{} // Reset the list
	players := getAllPlayers()
	if len(players) == 0 {
		return
	}
	for _, player := range players {
		meta := getMetadata(player)
		title := meta["title"]
		artist := meta["artist"]

		if isMusic(player, title, artist) {
			status := getPlaybackStatus(player)
			if status == "Playing" {
				log.Printf("Pausing music player %s: %s - %s", player, artist, title)
				cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Pause")
				_ = cmd.Run()
				sendNotification("Media Paused", fmt.Sprintf("Paused music: %s", title))
				playersToResume = append(playersToResume, player)
			} else {
				log.Printf("Music player %s is not playing (status: %s). Ignored.", player, status)
			}
		} else if title != "" {
			log.Printf("Non-music media detected on %s: %s. Not pausing.", player, title)
		}
	}
}

func playAllPlayers() {
	if len(playersToResume) == 0 {
		return
	}
	for _, player := range playersToResume {
		log.Printf("Resuming %s...", player)
		cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
		_ = cmd.Run()
	}
	playersToResume = []string{} // Clear the list after resuming
}

func checkAndPause(timings map[string]string, nowOverride ...time.Time) {
	now := time.Now()
	if len(nowOverride) > 0 {
		now = nowOverride[0]
	}
	prayers := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}

	inAnyWindow := false
	currentPrayerName := ""
	currentPrayerTime := ""

	for _, name := range prayers {
		tStr, ok := timings[name]
		if !ok {
			continue
		}

		t, err := time.Parse("15:04", tStr)
		if err != nil {
			log.Printf("Error parsing time for %s (%s): %v", name, tStr, err)
			continue
		}

		// Set time t to today
		pTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())

		// Stop 3 min before and 3 min after (as requested)
		start := pTime.Add(-3 * time.Minute)
		end := pTime.Add(3 * time.Minute)

		if now.After(start) && now.Before(end) {
			inAnyWindow = true
			currentPrayerName = name
			currentPrayerTime = tStr
			break
		}
	}

	if inAnyWindow {
		if !inPrayerBreak || lastHandledPrayer != currentPrayerName {
			// Window started or switched to a new prayer window
			inPrayerBreak = true
			lastHandledPrayer = currentPrayerName

			log.Printf("Entering prayer window for %s (%s). Pausing music...", currentPrayerName, currentPrayerTime)
			msg := fmt.Sprintf("It is time for %s (%s).", currentPrayerName, currentPrayerTime)
			sendNotification("Salat Break", msg)

			pauseAllPlayers()
		}
	} else {
		if inPrayerBreak {
			// We were in a prayer break but it has now ended
			log.Printf("Prayer window for %s ended at %s. Resuming music.", lastHandledPrayer, now.Format("15:04:05"))
			inPrayerBreak = false
			lastHandledPrayer = ""

			playAllPlayers()
			sendNotification("Salat Break Ended", "The prayer window has ended. Media playback resumed.")
		}
	}
}

func main() {
	testPause := flag.Bool("test-pause", false, "Run test: pause Spotify")
	testPlay := flag.Bool("test-play", false, "Run test: play Spotify")
	testNotify := flag.Bool("test-notify", false, "Run test: send notification")
	testLogic := flag.Bool("test-logic", false, "Run simulation to test pause/resume logic")
	flag.Parse()

	if *testPause {
		log.Println("Test: Pausing all media players...")
		pauseAllPlayers()
		return
	}
	if *testPlay {
		log.Println("Test: Playing all media players...")
		playAllPlayers()
		return
	}
	if *testNotify {
		log.Println("Test: Sending notification...")
		sendNotification("Salat Break Test", "This is a test notification for the Salat Break app.")
		return
	}

	if *testLogic {
		log.Println("Test: Running pause/resume logic simulation...")
		mockTimings := map[string]string{"Dhuhr": "12:00"}
		t1200, _ := time.Parse("15:04", "12:00")
		today := time.Now()
		baseTime := time.Date(today.Year(), today.Month(), today.Day(), t1200.Hour(), t1200.Minute(), 0, 0, today.Location())

		log.Println("--- Scenario 1: T-4 min (No window) ---")
		checkAndPause(mockTimings, baseTime.Add(-4*time.Minute))
		
		log.Println("--- Scenario 2: T-2 min (Entering window) ---")
		checkAndPause(mockTimings, baseTime.Add(-2*time.Minute))
		
		log.Println("--- Scenario 3: T+1 min (Still in window) ---")
		checkAndPause(mockTimings, baseTime.Add(1*time.Minute))
		
		log.Println("--- Scenario 4: T+4 min (Exiting window) ---")
		checkAndPause(mockTimings, baseTime.Add(4*time.Minute))
		
		log.Println("--- Simulation Complete ---")
		return
	}

	var loc *Location
	var lastLocCheck time.Time

	for {
		// Periodically check location (or if first run)
		if loc == nil || time.Since(lastLocCheck) > 15*time.Minute {
			newLoc, err := getBrowserLocation()
			if err != nil {
				log.Printf("Could not determine location: %v", err)
			} else if loc == nil || newLoc.City != loc.City || newLoc.Country != loc.Country {
				loc = newLoc
				log.Printf("Current location: %s, %s (Timezone: %s)", loc.City, loc.Country, loc.Timezone)
			}
			lastLocCheck = time.Now()
		}

		if loc != nil {
			pt, err := getPrayerTimes(loc)
			if err != nil {
				log.Printf("Error getting prayer times: %v", err)
			} else {
				checkAndPause(pt.Data.Timings)
			}
		} else {
			log.Printf("Waiting for valid location and internet to fetch data...")
		}
		
		// Check for prayer window every 30 seconds
		time.Sleep(30 * time.Second)
	}
}
