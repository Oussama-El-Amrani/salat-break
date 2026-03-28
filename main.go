package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

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

func getBrowserLocation() (*Location, error) {
	resp, err := http.Get("http://ip-api.com/json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var loc Location
	if err := json.NewDecoder(resp.Body).Decode(&loc); err != nil {
		return nil, err
	}
	return &loc, nil
}

func getPrayerTimes(loc *Location) (*PrayerTimes, error) {
	date := time.Now().Format("02-01-2006")
	url := fmt.Sprintf("http://api.aladhan.com/v1/timingsByCity/%s?city=%s&country=%s&method=2", date, loc.City, loc.Country)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var pt PrayerTimes
	if err := json.NewDecoder(resp.Body).Decode(&pt); err != nil {
		return nil, err
	}
	return &pt, nil
}

func pauseSpotify() {
	cmd := exec.Command("dbus-send", "--print-reply", "--dest=org.mpris.MediaPlayer2.spotify", "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Pause")
	err := cmd.Run()
	if err != nil {
		// Only log if it's not a "service not found" error (Spotify not running)
		log.Printf("Error pausing Spotify (maybe it's not running?): %v", err)
	} else {
		log.Println("Sent pause command to Spotify.")
	}
}

func playSpotify() {
	cmd := exec.Command("dbus-send", "--print-reply", "--dest=org.mpris.MediaPlayer2.spotify", "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
	err := cmd.Run()
	if err != nil {
		log.Printf("Error playing Spotify: %v", err)
	} else {
		log.Println("Sent play command to Spotify.")
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
			log.Printf("Current time %s is within window for %s (%s). Pausing Spotify...", now.Format("15:04:05"), name, tStr)
			pauseSpotify()
		}
	}
}

func main() {
	testPause := flag.Bool("test-pause", false, "Run test: pause Spotify")
	testPlay := flag.Bool("test-play", false, "Run test: play Spotify")
	flag.Parse()

	if *testPause {
		log.Println("Test: Pausing Spotify...")
		pauseSpotify()
		return
	}
	if *testPlay {
		log.Println("Test: Playing Spotify...")
		playSpotify()
		return
	}

	loc, err := getBrowserLocation()
	if err != nil {
		log.Fatalf("Error getting location: %v", err)
	}
	log.Printf("Detected location: %s, %s (Timezone: %s)", loc.City, loc.Country, loc.Timezone)

	// Update local timezone if needed? Go usually uses machine timezone.
	// loc.Timezone can be used to load location if current machine TZ is wrong.

	for {
		pt, err := getPrayerTimes(loc)
		if err != nil {
			log.Printf("Error getting prayer times: %v", err)
			time.Sleep(1 * time.Minute)
			continue
		}
		
		checkAndPause(pt.Data.Timings)
		
		// Check every 30 seconds
		time.Sleep(30 * time.Second)
	}
}
