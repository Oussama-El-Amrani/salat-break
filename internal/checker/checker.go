package checker

import (
	"fmt"
	"log"
	"time"

	"github.com/oussama_ib0/salat-break/internal/media"
	"github.com/oussama_ib0/salat-break/internal/notification"
)

type Service struct {
	mediaCtrl         *media.Controller
	notifier          *notification.Service
	inPrayerBreak     bool
	lastHandledPrayer string
}

func NewService(mediaCtrl *media.Controller, notifier *notification.Service) *Service {
	return &Service{
		mediaCtrl: mediaCtrl,
		notifier:  notifier,
	}
}

func (s *Service) CheckAndPause(timings map[string]string, nowOverride ...time.Time) {
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

		// Stop 3 min before and 3 min after
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
		if !s.inPrayerBreak || s.lastHandledPrayer != currentPrayerName {
			// Window started or switched to a new prayer window
			s.inPrayerBreak = true
			s.lastHandledPrayer = currentPrayerName

			log.Printf("Entering prayer window for %s (%s). Pausing music...", currentPrayerName, currentPrayerTime)
			msg := fmt.Sprintf("It is time for %s (%s).", currentPrayerName, currentPrayerTime)
			s.notifier.SendNotification("Salat Break", msg)

			s.mediaCtrl.PauseAllPlayers()
		}
	} else {
		if s.inPrayerBreak {
			// We were in a prayer break but it has now ended
			log.Printf("Prayer window for %s ended at %s. Resuming music.", s.lastHandledPrayer, now.Format("15:04:05"))
			s.inPrayerBreak = false
			s.lastHandledPrayer = ""

			s.mediaCtrl.PlayAllPlayers()
			s.notifier.SendNotification("Salat Break Ended", "The prayer window has ended. Media playback resumed.")
		}
	}
}
