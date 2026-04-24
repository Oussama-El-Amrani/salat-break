//go:build darwin

package notification

import (
	"fmt"
	"log"
	"os/exec"
)

func (s *Service) SendNotification(title, message string) {
	log.Printf("Sending macOS notification: %s - %s", title, message)

	script := fmt.Sprintf(`display notification "%s" with title "%s" subtitle "Salat Break"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	
	err := cmd.Run()
	if err != nil {
		log.Printf("Error sending macOS notification: %v", err)
	}
}
