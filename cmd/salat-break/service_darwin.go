//go:build darwin

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
)

func restartService() {
	log.Println("Restarting salat-break launchd agent to apply changes...")
	uid := os.Getuid()
	serviceName := "gui/%d/com.oussama.salat-break"
	fullServiceName := fmt.Sprintf(serviceName, uid)
	
	// kickstart -k restarts the service
	_ = exec.Command("launchctl", "kickstart", "-k", fullServiceName).Run()
}
