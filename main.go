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
	"strings"
	"time"
)

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
	resp, err := http.Get("http://ip-api.com/json")
	if err != nil {
		var cachedLoc Location
		if loadErr := loadCache("last_location.json", &cachedLoc); loadErr == nil {
			log.Printf("Offline or API error: Using cached location: %s, %s", cachedLoc.City, cachedLoc.Country)
			return &cachedLoc, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, err
	}
	
	_ = saveCache("last_location.json", loc)
	return &loc, nil
}

func getPrayerTimes(loc *Location) (*PrayerTimes, error) {
	date := time.Now().Format("02-01-2006")
	cacheKey := fmt.Sprintf("prayer_times_%s_%s_%s.json", strings.ToLower(loc.City), strings.ToLower(loc.Country), date)
	
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
	url := fmt.Sprintf("http://api.aladhan.com/v1/timingsByCity/%s?city=%s&country=%s&method=2", date, loc.City, loc.Country)
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
			// Extract service name from line like: string "org.mpris.MediaPlayer2.spotify"
			parts := strings.Split(line, "\"")
			if len(parts) >= 2 {
				players = append(players, parts[1])
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

func pauseAllPlayers() {
	players := getAllPlayers()
	if len(players) == 0 {
		return
	}
	for _, player := range players {
		meta := getMetadata(player)
		title := meta["title"]
		artist := meta["artist"]

		if isMusic(player, title, artist) {
			log.Printf("Pausing music player %s: %s - %s", player, artist, title)
			cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Pause")
			_ = cmd.Run()
			sendNotification("Media Paused", fmt.Sprintf("Paused music: %s", title))
		} else if title != "" {
			log.Printf("Non-music media detected on %s: %s. Not pausing.", player, title)
			sendNotification("Salat Reminder", fmt.Sprintf("Prayer time! (Playing: %s)", title))
		}
	}
}

func playAllPlayers() {
	players := getAllPlayers()
	if len(players) == 0 {
		return
	}
	for _, player := range players {
		log.Printf("Playing %s...", player)
		cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
		_ = cmd.Run()
	}
}

func checkAndPause(timings map[string]string) {
	now := time.Now()
	prayers := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}

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
		
		// Stop 2 min before and 3 min after
		start := pTime.Add(-2 * time.Minute)
		end := pTime.Add(3 * time.Minute)
		
		if now.After(start) && now.Before(end) {
			// Notify if not already sent for this prayer today
			lastSent, sent := notificationsSent[name]
			today := now.Truncate(24 * time.Hour)
			if !sent || lastSent.Before(today) {
				msg := fmt.Sprintf("It is time for %s (%s).", name, tStr)
				sendNotification("Salat Break", msg)
				notificationsSent[name] = now
			}

			log.Printf("Current time %s is within window for %s (%s). Checking players...", now.Format("15:04:05"), name, tStr)
			pauseAllPlayers()
		}
	}
}

func main() {
	testPause := flag.Bool("test-pause", false, "Run test: pause Spotify")
	testPlay := flag.Bool("test-play", false, "Run test: play Spotify")
	testNotify := flag.Bool("test-notify", false, "Run test: send notification")
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
