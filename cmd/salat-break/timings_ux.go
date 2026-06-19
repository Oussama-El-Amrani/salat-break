package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
	"github.com/Oussama-El-Amrani/salat-break/internal/location"
	"github.com/Oussama-El-Amrani/salat-break/internal/prayer"
)

// prayerDisplay holds display info for a single prayer.
type prayerDisplay struct {
	name         string
	timeStr      string
	time         time.Time
	status       string
	remainingStr string
}

func isTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

var (
	colorReset     = ""
	colorBold      = ""
	colorDim       = ""
	colorGreenBold = ""
	colorCyan      = ""
	colorYellow    = ""
	colorRedBold   = ""
	colorBlue      = ""
)

func initColors() {
	if isTTY() {
		colorReset     = "\033[0m"
		colorBold      = "\033[1m"
		colorDim       = "\033[90m"
		colorGreenBold = "\033[1;32m"
		colorCyan      = "\033[36m"
		colorYellow    = "\033[33m"
		colorRedBold   = "\033[1;31m"
		colorBlue      = "\033[34m"
	}
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func getTomorrowFajr(loc *location.Location, prayerSvc *prayer.Service, now time.Time) (string, bool) {
	tomorrow := now.AddDate(0, 0, 1)
	tomorrowDate := tomorrow.Format("02-01-2006")
	
	var cacheKey string
	if loc.Lat != 0 || loc.Lon != 0 {
		cacheKey = fmt.Sprintf("prayer_times_%.2f_%.2f_%s_m%d.json",
			loc.Lat, loc.Lon, tomorrowDate, prayerSvc.Method)
	} else {
		safeCity := cache.SanitizeName(loc.City)
		safeCountry := cache.SanitizeName(loc.Country)
		cacheKey = fmt.Sprintf("prayer_times_%s_%s_%s_m%d.json",
			strings.ToLower(safeCity), strings.ToLower(safeCountry), tomorrowDate, prayerSvc.Method)
	}

	var pt prayer.PrayerTimes
	if err := cache.Load(cacheKey, &pt); err == nil {
		if t, ok := pt.Data.Timings["Fajr"]; ok {
			return t, true
		}
	}
	return "", false
}

func humanizeSource(source string) string {
	switch source {
	case "wifi-beacondb":
		return "WiFi Geolocation (BeaconDB)"
	case "wifi-google":
		return "WiFi Geolocation (Google)"
	case "ip-consensus":
		return "IP Address Consensus"
	case "manual-coords":
		return "Manual Coordinates"
	case "manual-city-only":
		return "Manual Location Override"
	case "cached":
		return "Cached Location"
	default:
		if strings.HasPrefix(source, "ip-") {
			return fmt.Sprintf("IP Geolocation (%s)", strings.TrimPrefix(source, "ip-"))
		}
		return source
	}
}

func getMethodName(methodID int) string {
	switch methodID {
	case 0:
		return "Auto-detected"
	case 1:
		return "University of Islamic Sciences, Karachi"
	case 2:
		return "Islamic Society of North America (ISNA)"
	case 3:
		return "Muslim World League (MWL)"
	case 4:
		return "Umm al-Qura University, Makkah"
	case 5:
		return "Egyptian General Authority of Survey"
	case 7:
		return "Institute of Geophysics, University of Tehran"
	case 8:
		return "Gulf Region"
	case 9:
		return "Kuwait"
	case 10:
		return "Qatar"
	case 11:
		return "Majlis Ugama Islam Singapura, Singapore"
	case 12:
		return "Union des Organisations Islamiques de France (UOIF)"
	case 13:
		return "Diyanet İşleri Başkanlığı, Turkey"
	case 14:
		return "Spiritual Administration of Muslims of Russia"
	case 15:
		return "Moonsighting Committee Worldwide"
	case 16:
		return "UAE / Dubai"
	case 17:
		return "JAKIM, Malaysia"
	case 18:
		return "Tunisia"
	case 19:
		return "Algeria"
	case 20:
		return "KEMENAG, Indonesia"
	case 21:
		return "Morocco (Ministry of Habous and Islamic Affairs)"
	default:
		return fmt.Sprintf("Method ID %d", methodID)
	}
}

func printTimingsUX(loc *location.Location, prayerSvc *prayer.Service, timings *prayer.PrayerTimes) {
	initColors()
	now := time.Now()

	prayers := []string{"Fajr", "Dhuhr", "Asr", "Maghrib", "Isha"}
	var list []prayerDisplay

	for _, name := range prayers {
		tStr, ok := timings.Data.Timings[name]
		if !ok {
			continue
		}
		cleanTStr := tStr
		if parts := strings.Fields(tStr); len(parts) > 0 {
			cleanTStr = parts[0]
		}
		t, err := time.Parse("15:04", cleanTStr)
		if err != nil {
			continue
		}
		pTime := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
		list = append(list, prayerDisplay{
			name:    name,
			timeStr: tStr,
			time:    pTime,
		})
	}

	// Determine activeIndex & nextIndex
	activeIndex := -1
	for i, p := range list {
		start := p.time.Add(-3 * time.Minute)
		end := p.time.Add(3 * time.Minute)
		if now.After(start) && now.Before(end) {
			activeIndex = i
			break
		}
	}

	nextIndex := -1
	for i, p := range list {
		start := p.time.Add(-3 * time.Minute)
		if now.Before(start) {
			nextIndex = i
			break
		}
	}

	actualNextIndex := -1
	if activeIndex != -1 {
		if activeIndex < len(list)-1 {
			actualNextIndex = activeIndex + 1
		}
	} else {
		actualNextIndex = nextIndex
	}

	// Populate statuses and relative times
	for i := range list {
		p := &list[i]
		if i == activeIndex {
			p.status = "Active"
			end := p.time.Add(3 * time.Minute)
			p.remainingStr = fmt.Sprintf("ends in %s", formatDuration(end.Sub(now)))
		} else if i == actualNextIndex {
			p.status = "Next"
			p.remainingStr = fmt.Sprintf("in %s", formatDuration(p.time.Sub(now)))
		} else if (activeIndex != -1 && i < activeIndex) || (activeIndex == -1 && nextIndex != -1 && i < nextIndex) || (nextIndex == -1 && activeIndex == -1) {
			p.status = "Passed"
			p.remainingStr = fmt.Sprintf("%s ago", formatDuration(now.Sub(p.time)))
		} else {
			p.status = "Upcoming"
			p.remainingStr = fmt.Sprintf("in %s", formatDuration(p.time.Sub(now)))
		}
	}

	// Print main dashboard header
	fmt.Printf("\n%s🕌  SALAT BREAK TIMINGS%s\n", colorBold, colorReset)
	fmt.Printf("%s────────────────────────────────────────────────────────%s\n", colorCyan, colorReset)
	
	locPart := fmt.Sprintf("%s, %s", loc.City, loc.Country)
	if loc.IsManual {
		locPart += " (Manual Override)"
	} else {
		locPart += " (Auto-detected)"
	}
	fmt.Printf("%s📍 Location:  %s%s%s\n", colorCyan, colorReset, colorBold, locPart)
	fmt.Printf("%s📡 Source:    %s%s%s\n", colorCyan, colorReset, colorDim, fmt.Sprintf("%s (accuracy: %.0fm)", humanizeSource(loc.Source), loc.Accuracy))
	fmt.Printf("%s⚙️  Method:    %s%s%s\n", colorCyan, colorReset, colorDim, getMethodName(prayerSvc.Method))
	fmt.Printf("%s📅 Date:      %s%s (local time: %s%s)%s\n", colorCyan, colorReset, now.Format("Monday, 02 Jan 2006"), colorBold, now.Format("15:04:05"), colorReset)
	fmt.Printf("%s────────────────────────────────────────────────────────%s\n", colorCyan, colorReset)

	// Print table headers
	fmt.Printf("  %s%-10s %-10s %-12s %-18s%s\n", colorBold, "Prayer", "Time", "Status", "Remaining / Elapsed", colorReset)
	fmt.Printf("%s────────────────────────────────────────────────────────%s\n", colorCyan, colorReset)

	// Print table rows
	for _, p := range list {
		var indicator string
		var rowColor string

		switch p.status {
		case "Active":
			indicator = "●"
			rowColor = colorYellow
		case "Next":
			indicator = "➔"
			rowColor = colorGreenBold
		case "Passed":
			indicator = " "
			rowColor = colorDim
		default:
			indicator = " "
			rowColor = colorReset
		}

		fmt.Printf("%s%s  %-10s %-10s %-12s %-18s%s\n", 
			rowColor, indicator, p.name, p.timeStr, p.status, p.remainingStr, colorReset)
	}

	fmt.Printf("%s────────────────────────────────────────────────────────%s\n", colorCyan, colorReset)

	// Next Break details
	if activeIndex != -1 {
		activeP := list[activeIndex]
		startStr := activeP.time.Add(-3 * time.Minute).Format("15:04")
		endStr := activeP.time.Add(3 * time.Minute).Format("15:04")
		fmt.Printf("%s📢 Active Break: %s is currently active! Ends in %s (Window: %s - %s)%s\n",
			colorYellow, activeP.name, formatDuration(activeP.time.Add(3*time.Minute).Sub(now)), startStr, endStr, colorReset)
	}

	if actualNextIndex != -1 {
		nextP := list[actualNextIndex]
		startStr := nextP.time.Add(-3 * time.Minute).Format("15:04")
		endStr := nextP.time.Add(3 * time.Minute).Format("15:04")
		fmt.Printf("%s👉 Next Break:   %s in %s (Break window: %s - %s)%s\n",
			colorGreenBold, nextP.name, formatDuration(nextP.time.Add(-3*time.Minute).Sub(now)), startStr, endStr, colorReset)
	} else {
		// All of today's breaks have passed, resolve tomorrow's Fajr
		tomorrowFajrStr, ok := getTomorrowFajr(loc, prayerSvc, now)
		isEstimate := false
		if !ok {
			// fallback to today's Fajr
			for _, p := range list {
				if p.name == "Fajr" {
					tomorrowFajrStr = p.timeStr
					isEstimate = true
					break
				}
			}
		}

		if tomorrowFajrStr != "" {
			cleanFajrStr := tomorrowFajrStr
			if parts := strings.Fields(tomorrowFajrStr); len(parts) > 0 {
				cleanFajrStr = parts[0]
			}
			t, err := time.Parse("15:04", cleanFajrStr)
			if err == nil {
				tomorrow := now.AddDate(0, 0, 1)
				fajrTime := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
				breakStart := fajrTime.Add(-3 * time.Minute)
				breakEnd := fajrTime.Add(3 * time.Minute)

				estIndicator := ""
				if isEstimate {
					estIndicator = " ~"
				}

				fmt.Printf("%s👉 Next Break:   Fajr tomorrow in %s (Break window: %s - %s%s)%s\n",
					colorGreenBold, formatDuration(breakStart.Sub(now)), breakStart.Format("15:04"), breakEnd.Format("15:04"), estIndicator, colorReset)
			}
		}
	}
	fmt.Println()
}
