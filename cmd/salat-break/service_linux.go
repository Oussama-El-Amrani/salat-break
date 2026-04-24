//go:build linux

package main

import (
	"log"
	"os/exec"
)

func restartService() {
	log.Println("Restarting salat-break systemd service to apply changes...")
	_ = exec.Command("systemctl", "--user", "restart", "salat-break.service").Run()
}
