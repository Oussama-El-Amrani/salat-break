package media

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Notifier interface {
	SendNotification(title, message string)
}

type Controller struct {
	notifier        Notifier
	playersToResume []string
}

func NewController(notifier Notifier) *Controller {
	return &Controller{
		notifier: notifier,
	}
}

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

func (c *Controller) GetMetadata(player string) map[string]string {
	metadata := make(map[string]string)
	cmd := exec.Command("dbus-send", "--print-reply", "--session", "--dest="+player, "/org/mpris/MediaPlayer2", "org.freedesktop.DBus.Properties.Get", "string:org.mpris.MediaPlayer2.Player", "string:Metadata")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return metadata
	}

	lines := strings.Split(string(output), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "string \"xesam:title\"") || strings.Contains(line, "string \"xesam:artist\"") {
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

func (c *Controller) IsMusic(player string, title, artist string) bool {
	player = strings.ToLower(player)
	title = strings.ToLower(title)
	artist = strings.ToLower(artist)

	musicPlayers := []string{"spotify", "youtube_music", "rhythmbox", "clementine", "mpd", "audacious", "music"}
	for _, mp := range musicPlayers {
		if strings.Contains(player, mp) {
			return true
		}
	}

	musicKeywords := []string{"music", "song", "official video", "official audio", "lyrics", "cover", "remix", "album", "playlist", "feat."}
	for _, kw := range musicKeywords {
		if strings.Contains(title, kw) {
			return true
		}
	}

	return false
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
	c.playersToResume = []string{}
	players := c.GetAllPlayers()
	if len(players) == 0 {
		return
	}
	
	var pausedTitles []string
	for _, player := range players {
		meta := c.GetMetadata(player)
		title := meta["title"]
		artist := meta["artist"]

		if c.IsMusic(player, title, artist) {
			status := c.GetPlaybackStatus(player)
			if status == "Playing" {
				log.Printf("Pausing music player %s: %s - %s", player, artist, title)
				cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Pause")
				_ = cmd.Run()
				
				if title != "" {
					pausedTitles = append(pausedTitles, title)
				} else {
					pausedTitles = append(pausedTitles, player)
				}
				c.playersToResume = append(c.playersToResume, player)
			} else {
				log.Printf("Music player %s is not playing (status: %s). Ignored.", player, status)
			}
		} else if title != "" {
			log.Printf("Non-music media detected on %s: %s. Not pausing.", player, title)
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
		cmd := exec.Command("dbus-send", "--print-reply", "--dest="+player, "/org/mpris/MediaPlayer2", "org.mpris.MediaPlayer2.Player.Play")
		_ = cmd.Run()
	}
	c.playersToResume = []string{}
}
