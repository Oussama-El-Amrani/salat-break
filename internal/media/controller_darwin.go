//go:build darwin

package media

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func (c *Controller) GetAllPlayers() []string {
	var players []string
	
	// Common macOS media players
	apps := []string{"Music", "Spotify", "IINA", "VLC"}
	
	for _, app := range apps {
		// Check if the app is running
		cmd := exec.Command("pgrep", "-x", app)
		if err := cmd.Run(); err == nil {
			players = append(players, app)
		}
	}
	
	return players
}

func (c *Controller) GetMetadata(player string) map[string]string {
	metadata := make(map[string]string)
	
	var script string
	switch player {
	case "Music", "iTunes":
		script = `tell application "Music" to get {name, artist} of current track`
	case "Spotify":
		script = `tell application "Spotify" to get {name, artist} of current track`
	default:
		return metadata
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return metadata
	}

	parts := strings.Split(strings.TrimSpace(string(output)), ", ")
	if len(parts) >= 1 {
		metadata["title"] = parts[0]
	}
	if len(parts) >= 2 {
		metadata["artist"] = parts[1]
	}
	
	return metadata
}

func (c *Controller) GetPlaybackStatus(player string) string {
	var script string
	switch player {
	case "Music", "iTunes":
		script = `tell application "Music" to player state as string`
	case "Spotify":
		script = `tell application "Spotify" to player state as string`
	default:
		return ""
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	
	status := strings.TrimSpace(string(output))
	switch status {
	case "playing":
		return "Playing"
	case "paused":
		return "Paused"
	case "stopped":
		return "Stopped"
	default:
		return ""
	}
}

func (c *Controller) PauseAllPlayers() {
	c.playersToResume = []string{}
	players := c.GetAllPlayers()
	
	var pausedTitles []string
	for _, player := range players {
		status := c.GetPlaybackStatus(player)
		if status == "Playing" {
			meta := c.GetMetadata(player)
			title := meta["title"]
			artist := meta["artist"]

			if c.IsMusic(player, title, artist) {
				log.Printf("Pausing music player %s: %s - %s", player, artist, title)
				script := fmt.Sprintf(`tell application "%s" to pause`, player)
				_ = exec.Command("osascript", "-e", script).Run()
				
				if title != "" {
					pausedTitles = append(pausedTitles, title)
				} else {
					pausedTitles = append(pausedTitles, player)
				}
				c.playersToResume = append(c.playersToResume, player)
			}
		}
	}

	if len(pausedTitles) > 0 {
		msg := fmt.Sprintf("Paused: %s", strings.Join(pausedTitles, ", "))
		if len(pausedTitles) > 2 {
			msg = fmt.Sprintf("Paused %d media players", len(pausedTitles))
		}
		c.notifier.SendNotification("Media Paused", msg)
	}
}

func (c *Controller) PlayAllPlayers() {
	if len(c.playersToResume) == 0 {
		return
	}
	for _, player := range c.playersToResume {
		log.Printf("Resuming %s...", player)
		script := fmt.Sprintf(`tell application "%s" to play`, player)
		_ = exec.Command("osascript", "-e", script).Run()
	}
	c.playersToResume = []string{}
}
