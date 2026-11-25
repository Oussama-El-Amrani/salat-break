//go:build linux

package media

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func (c *Controller) GetAllPlayers() []string {
	cmd := exec.Command("dbus-send", "--session", "--dest=org.freedesktop.DBus", "--type=method_call", "--print-reply", "/org/freedesktop/DBus", "org.freedesktop.DBus.ListNames")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error listing DBus names: %v", err)
		return nil
	}

	var players []string
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "org.mpris.MediaPlayer2.") {
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

func extractDBusDictValue(lines []string, keyLineIdx int) string {
	for offset := 1; offset <= 3 && keyLineIdx+offset < len(lines); offset++ {
		valLine := strings.TrimSpace(lines[keyLineIdx+offset])
		if valLine == ")" || strings.HasPrefix(valLine, "dict entry(") {
			break
		}
		parts := strings.Split(valLine, "\"")
		if len(parts) >= 2 {
			return parts[1]
		}
	}
	return ""
}

func (c *Controller) GetMetadata(player string) map[string]string {
	metadata := make(map[string]string)
	cmd := exec.Command("dbus-send", "--print-reply", "--session", "--dest="+player, "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Properties.Get", "string:org.mpris.MediaPlayer2.Player", "string:Metadata")
	output, err := cmd.CombinedOutput()
	if err != nil {
		metadata["provider"] = ResolveProviderName(player, metadata)
		return metadata
	}

	outputStr := string(output)
	lowerOut := strings.ToLower(outputStr)
	if strings.Contains(lowerOut, "spotify") || strings.Contains(lowerOut, "scdn.co") {
		metadata["is_spotify"] = "true"
	}

	lines := strings.Split(outputStr, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "string \"xesam:title\""):
			if v := extractDBusDictValue(lines, i); v != "" {
				metadata["title"] = v
			}
		case strings.Contains(line, "string \"xesam:artist\""):
			if v := extractDBusDictValue(lines, i); v != "" {
				metadata["artist"] = v
			}
		case strings.Contains(line, "string \"xesam:album\""):
			if v := extractDBusDictValue(lines, i); v != "" {
				metadata["album"] = v
			}
		case strings.Contains(line, "string \"mpris:artUrl\""):
			if v := extractDBusDictValue(lines, i); v != "" {
				metadata["artUrl"] = v
			}
		case strings.Contains(line, "string \"xesam:url\""):
			if v := extractDBusDictValue(lines, i); v != "" {
				metadata["url"] = v
			}
		}
	}

	metadata["provider"] = ResolveProviderName(player, metadata)
	return metadata
}

func (c *Controller) GetPlaybackStatus(player string) string {
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

func (c *Controller) PauseAllPlayers() {
	c.pauseAllPlayers(false)
}

func (c *Controller) EnforceSilence() {
	c.pauseAllPlayers(true)
}

func (c *Controller) pauseAllPlayers(accumulate bool) {
	if !accumulate {
		c.playersToResume = []string{}
	}
	players := c.GetAllPlayers()
	if len(players) == 0 {
		return
	}

	var pausedTitles []string
	for _, player := range players {
		meta := c.GetMetadata(player)
		title := meta["title"]
		artist := meta["artist"]

		if c.IsProviderAllowed(player, meta) {
			log.Printf("Provider %s (%s) is allowed during Salat slot. Skipping pause.", ResolveProviderName(player, meta), player)
			continue
		}

		if c.IsMusic(player, meta) {
			status := c.GetPlaybackStatus(player)
			if status == "Playing" {
				log.Printf("Pausing media provider %s: %s - %s", player, artist, title)
				cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Pause")
				_ = cmd.Run()

				if title != "" {
					pausedTitles = append(pausedTitles, title)
				} else {
					pausedTitles = append(pausedTitles, ResolveProviderName(player, meta))
				}

				// Only add to playersToResume if not already present
				found := false
				for _, p := range c.playersToResume {
					if p == player {
						found = true
						break
					}
				}
				if !found {
					c.playersToResume = append(c.playersToResume, player)
				}
			} else {
				log.Printf("Media provider %s is not playing (status: %s). Ignored.", player, status)
			}
		} else if title != "" {
			log.Printf("Non-music browser media detected on %s: %s. Not pausing.", player, title)
		}
	}

	if len(pausedTitles) > 0 {
		msg := fmt.Sprintf("Paused: %s", strings.Join(pausedTitles, ", "))
		if len(pausedTitles) > 2 {
			msg = fmt.Sprintf("Paused %d media players", len(pausedTitles))
		}
		if accumulate {
			c.notifier.SendNotification("Media Paused during prayer window", msg)
		} else {
			c.notifier.SendNotification("Media Paused", msg)
		}
	}
}

func (c *Controller) PlayAllPlayers() {
	if len(c.playersToResume) == 0 {
		return
	}
	for _, player := range c.playersToResume {
		log.Printf("Resuming %s...", player)
		cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
		_ = cmd.Run()
	}
	c.playersToResume = []string{}
}

func (c *Controller) ForcePlayAllPlayers() {
	players := c.GetAllPlayers()
	for _, p := range players {
		log.Printf("Test: Resuming player %s", p)
		cmd := exec.Command("dbus-send", "--print-reply", "--dest="+p, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
		_ = cmd.Run()
	}
}
