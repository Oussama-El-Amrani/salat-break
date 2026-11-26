package media

import (
	"context"
	"strings"
	"sync"
)

type Notifier interface {
	SendNotification(title, message string)
}

type Controller struct {
	mu               sync.Mutex
	cfgMu            sync.RWMutex
	notifier         Notifier
	playersToResume  []string
	allowedProviders []string
	silenceCancel    context.CancelFunc
}

func NewController(notifier Notifier) *Controller {
	return &Controller{
		notifier:         notifier,
		allowedProviders: []string{},
	}
}

// SetAllowedProviders configures the list of media providers allowed to keep playing during Salat slots.
func (c *Controller) SetAllowedProviders(providers []string) {
	var normalized []string
	for _, raw := range providers {
		for _, p := range SplitProviderInput(raw) {
			if !containsString(normalized, p) {
				normalized = append(normalized, p)
			}
		}
	}
	c.cfgMu.Lock()
	c.allowedProviders = normalized
	c.cfgMu.Unlock()
}

// GetAllowedProviders returns a copy of currently configured allowed providers.
func (c *Controller) GetAllowedProviders() []string {
	c.cfgMu.RLock()
	defer c.cfgMu.RUnlock()
	out := make([]string, len(c.allowedProviders))
	copy(out, c.allowedProviders)
	return out
}

// DetectWebMusicProvider inspects track metadata (URL, artUrl, explicit provider flags)
// to identify web-based music streaming services running inside a browser or PWA.
func DetectWebMusicProvider(metadata map[string]string) string {
	if metadata == nil {
		return ""
	}
	if p := NormalizeProviderName(metadata["provider"]); p != "" && !isBrowserName(p) {
		return p
	}
	if metadata["is_spotify"] == "true" {
		return "spotify"
	}

	combined := strings.ToLower(metadata["url"] + " " + metadata["artUrl"])
	webProviders := []struct {
		provider   string
		indicators []string
	}{
		{"spotify", []string{"spotify.com", "scdn.co", "spotify:"}},
		{"youtube-music", []string{"music.youtube.com"}},
		{"soundcloud", []string{"soundcloud.com", "sndcdn.com"}},
		{"deezer", []string{"deezer.com", "dzcdn.net"}},
		{"tidal", []string{"tidal.com"}},
		{"apple-music", []string{"music.apple.com"}},
		{"anghami", []string{"anghami.com"}},
		{"amazon-music", []string{"music.amazon."}},
		{"qobuz", []string{"qobuz.com"}},
		{"bandcamp", []string{"bandcamp.com", "bcbits.com"}},
		{"pandora", []string{"pandora.com"}},
	}

	for _, wp := range webProviders {
		for _, ind := range wp.indicators {
			if strings.Contains(combined, ind) {
				return wp.provider
			}
		}
	}

	return ""
}

// ResolveProviderName determines the effective provider name for a player and its metadata.
// For standalone apps (e.g., VLC, Spotify, MPV, Rhythmbox), it returns the app's normalized name.
// For browsers playing a web music service (e.g., Spotify PWA, YouTube Music), it returns that service's name.
func ResolveProviderName(player string, metadata map[string]string) string {
	if webProv := DetectWebMusicProvider(metadata); webProv != "" {
		return webProv
	}
	return NormalizeProviderName(player)
}

func isBrowserName(norm string) bool {
	browsers := []string{
		"chromium",
		"chrome",
		"google-chrome",
		"firefox",
		"brave",
		"brave-browser",
		"msedge",
		"microsoft-edge",
		"edge",
		"safari",
		"opera",
		"vivaldi",
		"zen",
		"epiphany",
		"librewolf",
		"waterfox",
		"floorp",
		"thorium",
		"arc",
		"plasma-browser-integration",
	}
	for _, b := range browsers {
		if norm == b || strings.HasPrefix(norm, b+"-") {
			return true
		}
	}
	return false
}

func isBrowserPlayer(player string) bool {
	norm := NormalizeProviderName(player)
	return isBrowserName(norm)
}

// IsProviderAllowed returns true if the given player or its resolved music provider
// has been toggled/allowed to continue playing during Salat break slots.
func (c *Controller) IsProviderAllowed(player string, metadata map[string]string) bool {
	allowedList := c.GetAllowedProviders()
	if len(allowedList) == 0 {
		return false
	}

	var candidates []string
	addCandidate := func(val string) {
		if val != "" && !containsString(candidates, val) {
			candidates = append(candidates, val)
		}
	}

	addCandidate(NormalizeProviderName(player))
	addCandidate(ResolveProviderName(player, metadata))
	if metadata != nil {
		addCandidate(NormalizeProviderName(metadata["provider"]))
		if metadata["is_spotify"] == "true" {
			addCandidate("spotify")
		}
	}

	// Also add segment tokens from bus/virtual player names (e.g., browser name in Web-Spotify-Chrome)
	trimmed := strings.ToLower(strings.TrimSpace(player))
	trimmed = strings.TrimPrefix(trimmed, "org.mpris.mediaplayer2.")
	if idx := strings.Index(trimmed, ".instance"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	for _, seg := range strings.FieldsFunc(trimmed, func(r rune) bool {
		return r == '.' || r == '-' || r == '_'
	}) {
		if seg != "" && seg != "web" && seg != "org" && seg != "com" && seg != "io" && seg != "app" {
			addCandidate(seg)
		}
	}

	for _, allowed := range allowedList {
		if containsString(candidates, allowed) {
			return true
		}
	}
	return false
}

// IsMusic determines whether a detected media player should be controlled during prayer breaks.
// Any standalone media player (e.g., VLC, Spotify, MPV, Rhythmbox, Amberol, Strawberry, Music.app, VOX)
// is supported automatically. Generic web browsers are controlled only when playing a web music
// provider (Spotify, YouTube Music, SoundCloud, Deezer, Tidal, etc.) or matching music metadata,
// avoiding interruptions to non-music browser videos (like tutorials).
func (c *Controller) IsMusic(player string, metadata map[string]string) bool {
	if strings.TrimSpace(player) == "" {
		return false
	}

	// Web music providers (Spotify Web, YouTube Music, SoundCloud, Deezer, Tidal, etc.)
	if DetectWebMusicProvider(metadata) != "" {
		return true
	}

	// Any standalone media/music player (non-browser) is supported directly (VLC, MPV, Spotify, etc.)
	if !isBrowserPlayer(player) {
		return true
	}

	// For generic browser tabs, check title keywords to distinguish music from tutorials/lectures
	title := strings.ToLower(metadata["title"])
	musicKeywords := []string{
		"music", "song", "official video", "official audio",
		"lyrics", "cover", "remix", "album", "playlist", "feat.",
	}
	for _, kw := range musicKeywords {
		if strings.Contains(title, kw) {
			return true
		}
	}

	return false
}
