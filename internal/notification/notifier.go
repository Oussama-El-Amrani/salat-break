package notification

import (
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"time"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
)

type Service struct {
	timeout            int
	clearDelay         int
	lastNotificationID uint32
}

func NewService(timeout, clearDelay int) *Service {
	s := &Service{
		timeout:    timeout,
		clearDelay: clearDelay,
	}
	// Initial load of last ID
	_ = cache.Load("last_notification_id.json", &s.lastNotificationID)
	return s
}

func (s *Service) SendNotification(title, message string) {
	log.Printf("Sending notification: %s - %s (timeout: %dms, clear: %dms)", title, message, s.timeout, s.clearDelay)

	cmd := exec.Command("gdbus", "call", "--session", 
		"--dest=org.freedesktop.Notifications", 
		"--object-path=/org/freedesktop/Notifications", 
		"--method=org.freedesktop.Notifications.Notify", 
		"Salat Break", fmt.Sprintf("uint32 %d", s.lastNotificationID), "appointment-soon", 
		title, message, "[]", "{}", fmt.Sprintf("int32 %d", s.timeout))

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error sending DBus notification: %v (output: %s)", err, string(output))
		return
	}

	re := regexp.MustCompile(`uint32 (\d+)`)
	matches := re.FindStringSubmatch(string(output))
	if len(matches) >= 2 {
		var id uint32
		fmt.Sscanf(matches[1], "%d", &id)
		s.lastNotificationID = id
		
		_ = cache.Save("last_notification_id.json", id)

		// Start a goroutine to clear from tray after clearDelay
		go func(notifID uint32, delay int) {
			time.Sleep(time.Duration(delay) * time.Millisecond)
			log.Printf("Automatically clearing notification ID %d from tray...", notifID)
			closeCmd := exec.Command("gdbus", "call", "--session", 
				"--dest=org.freedesktop.Notifications", 
				"--object-path=/org/freedesktop/Notifications", 
				"--method=org.freedesktop.Notifications.CloseNotification", 
				fmt.Sprintf("uint32 %d", notifID))
			_ = closeCmd.Run()
		}(id, s.clearDelay)
	}
}
