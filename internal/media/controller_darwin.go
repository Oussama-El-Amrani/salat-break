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

	// Check web browsers for active Spotify tab/PWA
	browsers := []struct {
		appName    string
		playerType string
	}{
		{"Google Chrome", "Spotify-Web-Chrome"},
		{"Brave Browser", "Spotify-Web-Brave"},
		{"Microsoft Edge", "Spotify-Web-Edge"},
		{"Safari", "Spotify-Web-Safari"},
	}

	for _, b := range browsers {
		// Only check if browser is running
		cmd := exec.Command("pgrep", "-x", b.appName)
		if err := cmd.Run(); err == nil {
			if c.hasSpotifyTab(b.appName) {
				players = append(players, b.playerType)
			}
		}
	}
	
	return players
}

func (c *Controller) hasSpotifyTab(appName string) bool {
	var script string
	if appName == "Safari" {
		script = `
tell application "Safari"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "open.spotify.com" then
				return "true"
			end if
		end repeat
	end repeat
end tell
return "false"
`
	} else {
		script = fmt.Sprintf(`
tell application "%s"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "open.spotify.com" then
				return "true"
			end if
		end repeat
	end repeat
end tell
return "false"
`, appName)
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func (c *Controller) getAppNameFromVirtualPlayer(player string) string {
	switch player {
	case "Spotify-Web-Chrome":
		return "Google Chrome"
	case "Spotify-Web-Brave":
		return "Brave Browser"
	case "Spotify-Web-Edge":
		return "Microsoft Edge"
	case "Spotify-Web-Safari":
		return "Safari"
	default:
		return ""
	}
}

func (c *Controller) executeJSInSpotifyTab(appName string, jsCode string) string {
	var script string
	// Escape backslashes and double quotes for AppleScript string literals
	escapedJS := strings.ReplaceAll(jsCode, `\`, `\\`)
	escapedJS = strings.ReplaceAll(escapedJS, `"`, `\"`)

	if appName == "Safari" {
		script = fmt.Sprintf(`
tell application "Safari"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "open.spotify.com" then
				tell t
					return do javascript "%s"
				end tell
			end if
		end repeat
	end repeat
end tell
`, escapedJS)
	} else {
		script = fmt.Sprintf(`
tell application "%s"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "open.spotify.com" then
				tell t
					return execute javascript "%s"
				end tell
			end if
		end repeat
	end repeat
end tell
`, appName, escapedJS)
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func (c *Controller) GetMetadata(player string) map[string]string {
	metadata := make(map[string]string)
	
	if strings.HasPrefix(player, "Spotify-Web-") {
		appName := c.getAppNameFromVirtualPlayer(player)
		metadata["is_spotify"] = "true"
		metadata["title"] = "Spotify Web Player"
		
		jsTitle := `(function() {
			let titleEl = document.querySelector('[data-testid="track-info-metadata-title"]') || document.querySelector('title');
			return titleEl ? titleEl.innerText : '';
		})()`
		jsArtist := `(function() {
			let artistEl = document.querySelector('[data-testid="track-info-metadata-artists"]') || document.querySelector('[data-testid="track-info-metadata-artist"]');
			return artistEl ? artistEl.innerText : '';
		})()`

		title := c.executeJSInSpotifyTab(appName, jsTitle)
		artist := c.executeJSInSpotifyTab(appName, jsArtist)

		if title != "" {
			metadata["title"] = title
		}
		if artist != "" {
			metadata["artist"] = artist
		}
		return metadata
	}

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
	if strings.HasPrefix(player, "Spotify-Web-") {
		appName := c.getAppNameFromVirtualPlayer(player)
		jsStatus := `(function() {
			let pauseBtn = document.querySelector('[data-testid="control-button-pause"]') || document.querySelector('button[title*="Pause"]');
			let playBtn = document.querySelector('[data-testid="control-button-play"]') || document.querySelector('button[title*="Play"]');
			if (pauseBtn) return 'playing';
			if (playBtn) return 'paused';
			
			let audios = document.querySelectorAll('audio, video');
			for (let a of audios) {
				if (!a.paused) return 'playing';
			}
			return 'paused';
		})()`
		status := c.executeJSInSpotifyTab(appName, jsStatus)
		if status == "playing" {
			return "Playing"
		}
		return "Paused"
	}

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

			if c.IsMusic(player, meta) {
				log.Printf("Pausing music player %s: %s - %s", player, artist, title)
				if strings.HasPrefix(player, "Spotify-Web-") {
					appName := c.getAppNameFromVirtualPlayer(player)
					jsPause := `(function() {
						let pauseBtn = document.querySelector('[data-testid="control-button-pause"]') || document.querySelector('button[title*="Pause"]');
						if (pauseBtn) {
							pauseBtn.click();
						} else {
							let audios = document.querySelectorAll('audio, video');
							for (let a of audios) {
								if (!a.paused) a.pause();
							}
						}
					})()`
					_ = c.executeJSInSpotifyTab(appName, jsPause)
				} else {
					script := fmt.Sprintf(`tell application "%s" to pause`, player)
					_ = exec.Command("osascript", "-e", script).Run()
				}
				
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
		if strings.HasPrefix(player, "Spotify-Web-") {
			appName := c.getAppNameFromVirtualPlayer(player)
			jsPlay := `(function() {
				let playBtn = document.querySelector('[data-testid="control-button-play"]') || document.querySelector('button[title*="Play"]');
				if (playBtn) {
					playBtn.click();
				} else {
					let audios = document.querySelectorAll('audio, video');
					for (let a of audios) {
						if (a.paused) a.play();
					}
				}
			})()`
			_ = c.executeJSInSpotifyTab(appName, jsPlay)
		} else {
			script := fmt.Sprintf(`tell application "%s" to play`, player)
			_ = exec.Command("osascript", "-e", script).Run()
		}
	}
	c.playersToResume = []string{}
}
