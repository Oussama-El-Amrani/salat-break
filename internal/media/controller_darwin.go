//go:build darwin

package media

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type darwinWebProvider struct {
	provider string
	domain   string
}

var macOSWebProviders = []darwinWebProvider{
	{"spotify", "open.spotify.com"},
	{"youtube-music", "music.youtube.com"},
	{"soundcloud", "soundcloud.com"},
	{"deezer", "deezer.com"},
	{"tidal", "tidal.com"},
	{"apple-music", "music.apple.com"},
	{"anghami", "anghami.com"},
}

type darwinBrowser struct {
	appName string
	slug    string
}

var macOSBrowsers = []darwinBrowser{
	{"Google Chrome", "Chrome"},
	{"Brave Browser", "Brave"},
	{"Microsoft Edge", "Edge"},
	{"Safari", "Safari"},
	{"Arc", "Arc"},
}

func (c *Controller) GetAllPlayers() []string {
	var players []string

	// Common macOS media players
	apps := []string{"Music", "Spotify", "VLC", "VOX", "IINA", "TIDAL", "Deezer", "Audirvana", "Swinsian", "Clementine", "Strawberry"}

	for _, app := range apps {
		cmd := exec.Command("pgrep", "-x", app)
		if err := cmd.Run(); err == nil {
			players = append(players, app)
		}
	}

	// Check web browsers for active music provider tabs/PWAs
	for _, b := range macOSBrowsers {
		cmd := exec.Command("pgrep", "-x", b.appName)
		if err := cmd.Run(); err == nil {
			for _, wp := range macOSWebProviders {
				if c.hasMusicProviderTab(b.appName, wp.domain) {
					players = append(players, fmt.Sprintf("Web-%s-%s", wp.provider, b.slug))
				}
			}
		}
	}

	return players
}

func parseVirtualWebPlayer(player string) (appName string, domain string, provider string, ok bool) {
	if strings.HasPrefix(player, "Spotify-Web-") {
		slug := strings.TrimPrefix(player, "Spotify-Web-")
		for _, b := range macOSBrowsers {
			if b.slug == slug {
				return b.appName, "open.spotify.com", "spotify", true
			}
		}
		return "", "", "", false
	}

	if !strings.HasPrefix(player, "Web-") {
		return "", "", "", false
	}

	rest := strings.TrimPrefix(player, "Web-")
	lastDash := strings.LastIndex(rest, "-")
	if lastDash <= 0 {
		return "", "", "", false
	}
	prov := rest[:lastDash]
	slug := rest[lastDash+1:]

	for _, b := range macOSBrowsers {
		if b.slug == slug {
			appName = b.appName
			break
		}
	}
	for _, wp := range macOSWebProviders {
		if wp.provider == prov {
			domain = wp.domain
			provider = wp.provider
			break
		}
	}
	if appName == "" || domain == "" {
		return "", "", "", false
	}
	return appName, domain, provider, true
}

func (c *Controller) hasMusicProviderTab(appName, domain string) bool {
	script := fmt.Sprintf(`
tell application "%s"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "%s" then
				return "true"
			end if
		end repeat
	end repeat
end tell
return "false"
`, appName, domain)

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

func (c *Controller) executeJSInTab(appName, domain, jsCode string) string {
	escapedJS := strings.ReplaceAll(jsCode, `\`, `\\`)
	escapedJS = strings.ReplaceAll(escapedJS, `"`, `\"`)

	var script string
	if appName == "Safari" {
		script = fmt.Sprintf(`
tell application "Safari"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "%s" then
				tell t
					return do javascript "%s"
				end tell
			end if
		end repeat
	end repeat
end tell
`, domain, escapedJS)
	} else {
		script = fmt.Sprintf(`
tell application "%s"
	repeat with w in windows
		repeat with t in tabs of w
			if URL of t contains "%s" then
				tell t
					return execute javascript "%s"
				end tell
			end if
		end repeat
	end repeat
end tell
`, appName, domain, escapedJS)
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

	if appName, domain, prov, ok := parseVirtualWebPlayer(player); ok {
		metadata["provider"] = prov
		if prov == "spotify" {
			metadata["is_spotify"] = "true"
		}
		metadata["title"] = prov

		jsTitle := `(function() {
			if (navigator.mediaSession && navigator.mediaSession.metadata && navigator.mediaSession.metadata.title) {
				return navigator.mediaSession.metadata.title;
			}
			let titleEl = document.querySelector('[data-testid="track-info-metadata-title"]') || document.querySelector('title');
			return titleEl ? titleEl.innerText : '';
		})()`
		jsArtist := `(function() {
			if (navigator.mediaSession && navigator.mediaSession.metadata && navigator.mediaSession.metadata.artist) {
				return navigator.mediaSession.metadata.artist;
			}
			let artistEl = document.querySelector('[data-testid="track-info-metadata-artists"]') || document.querySelector('[data-testid="track-info-metadata-artist"]');
			return artistEl ? artistEl.innerText : '';
		})()`

		title := c.executeJSInTab(appName, domain, jsTitle)
		artist := c.executeJSInTab(appName, domain, jsArtist)

		if title != "" {
			metadata["title"] = title
		}
		if artist != "" {
			metadata["artist"] = artist
		}
		return metadata
	}

	metadata["provider"] = NormalizeProviderName(player)

	var script string
	switch player {
	case "Music", "iTunes":
		script = `tell application "Music" to get {name, artist} of current track`
	case "Spotify":
		script = `tell application "Spotify" to get {name, artist} of current track`
	case "VLC":
		script = `tell application "VLC" to get name of current item`
	case "VOX":
		script = `tell application "VOX" to get {track, artist}`
	default:
		return metadata
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return metadata
	}

	trimmed := strings.TrimSpace(string(output))
	if player == "VLC" {
		if trimmed != "" {
			metadata["title"] = trimmed
		}
		return metadata
	}

	parts := strings.Split(trimmed, ", ")
	if len(parts) >= 1 && parts[0] != "" {
		metadata["title"] = parts[0]
	}
	if len(parts) >= 2 && parts[1] != "" {
		metadata["artist"] = parts[1]
	}

	return metadata
}

func (c *Controller) GetPlaybackStatus(player string) string {
	if appName, domain, _, ok := parseVirtualWebPlayer(player); ok {
		jsStatus := `(function() {
			if (navigator.mediaSession && navigator.mediaSession.playbackState === 'playing') {
				return 'playing';
			}
			let audios = document.querySelectorAll('audio, video');
			for (let a of audios) {
				if (!a.paused) return 'playing';
			}
			let pauseBtn = document.querySelector('[data-testid="control-button-pause"]') || document.querySelector('button[title*="Pause"]');
			if (pauseBtn) return 'playing';
			return 'paused';
		})()`
		status := c.executeJSInTab(appName, domain, jsStatus)
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
	case "VLC":
		script = `tell application "VLC" to playing as string`
	case "VOX":
		script = `tell application "VOX" to player state as string`
	default:
		script = fmt.Sprintf(`tell application "%s" to player state as string`, player)
	}

	cmd := exec.Command("osascript", "-e", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}

	status := strings.ToLower(strings.TrimSpace(string(output)))
	switch status {
	case "playing", "true", "1":
		return "Playing"
	case "paused", "false", "0":
		return "Paused"
	case "stopped":
		return "Stopped"
	default:
		return ""
	}
}

func pauseMacOSPlayer(c *Controller, player string) {
	if appName, domain, _, ok := parseVirtualWebPlayer(player); ok {
		jsPause := `(function() {
			let pausedAny = false;
			let audios = document.querySelectorAll('audio, video');
			for (let a of audios) {
				if (!a.paused) { a.pause(); pausedAny = true; }
			}
			if (!pausedAny) {
				let pauseBtn = document.querySelector('[data-testid="control-button-pause"]') || document.querySelector('button[title*="Pause"]');
				if (pauseBtn) pauseBtn.click();
			}
		})()`
		_ = c.executeJSInTab(appName, domain, jsPause)
		return
	}

	var script string
	if player == "VLC" {
		// In macOS VLC's AppleScript dictionary, `play` toggles play/pause.
		script = `tell application "VLC" to if playing then play`
	} else {
		script = fmt.Sprintf(`tell application "%s" to pause`, player)
	}
	_ = exec.Command("osascript", "-e", script).Run()
}

func playMacOSPlayer(c *Controller, player string) {
	if appName, domain, _, ok := parseVirtualWebPlayer(player); ok {
		jsPlay := `(function() {
			let playedAny = false;
			let audios = document.querySelectorAll('audio, video');
			for (let a of audios) {
				if (a.paused) { a.play(); playedAny = true; }
			}
			if (!playedAny) {
				let playBtn = document.querySelector('[data-testid="control-button-play"]') || document.querySelector('button[title*="Play"]');
				if (playBtn) playBtn.click();
			}
		})()`
		_ = c.executeJSInTab(appName, domain, jsPlay)
		return
	}

	var script string
	if player == "VLC" {
		script = `tell application "VLC" to if not playing then play`
	} else {
		script = fmt.Sprintf(`tell application "%s" to play`, player)
	}
	_ = exec.Command("osascript", "-e", script).Run()
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

	var pausedTitles []string
	for _, player := range players {
		meta := c.GetMetadata(player)
		if c.IsProviderAllowed(player, meta) {
			log.Printf("Provider %s (%s) is allowed during Salat slot. Skipping pause.", ResolveProviderName(player, meta), player)
			continue
		}

		status := c.GetPlaybackStatus(player)
		if status == "Playing" {
			title := meta["title"]
			artist := meta["artist"]

			if c.IsMusic(player, meta) {
				log.Printf("Pausing media provider %s: %s - %s", player, artist, title)
				pauseMacOSPlayer(c, player)

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
			}
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
		playMacOSPlayer(c, player)
	}
	c.playersToResume = []string{}
}

func (c *Controller) ForcePlayAllPlayers() {
	players := c.GetAllPlayers()
	for _, player := range players {
		log.Printf("Test: Resuming player %s", player)
		playMacOSPlayer(c, player)
	}
}
