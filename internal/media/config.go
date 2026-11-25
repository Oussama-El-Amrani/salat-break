package media

import (
	"strings"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
)

const MediaConfigFile = "media_config.json"

// Config holds persistent media controller preferences.
type Config struct {
	AllowedProviders []string `json:"allowed_providers"`
}

// NormalizeProviderName canonicalizes a user-supplied or MPRIS/OS provider identifier.
// Examples:
//   - "VLC" -> "vlc"
//   - "org.mpris.MediaPlayer2.vlc.instance1234" -> "vlc"
//   - "org.mpris.MediaPlayer2.io.bassi.Amberol" -> "amberol"
//   - "Music.app" -> "music"
//   - "youtube_music" -> "youtube-music"
func NormalizeProviderName(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	if s == "" {
		return ""
	}

	s = strings.TrimPrefix(s, "org.mpris.mediaplayer2.")

	if idx := strings.Index(s, ".instance"); idx != -1 {
		s = s[:idx]
	}

	s = strings.TrimSuffix(s, ".app")

	// Handle macOS virtual web player identifiers (e.g., "Web-Spotify-Chrome" or "Spotify-Web-Chrome")
	if strings.HasPrefix(s, "spotify-web-") {
		return "spotify"
	}
	if strings.HasPrefix(s, "web-") {
		parts := strings.Split(s, "-")
		if len(parts) >= 3 {
			return strings.Join(parts[1:len(parts)-1], "-")
		}
	}

	// For reverse-DNS identifiers (e.g. "io.bassi.amberol", "org.videolan.vlc"), extract the app name
	if strings.Contains(s, ".") {
		parts := strings.Split(s, ".")
		last := parts[len(parts)-1]
		if last == "client" && len(parts) >= 2 {
			// e.g. "com.spotify.client" -> "spotify"
			s = parts[len(parts)-2]
		} else if last != "" {
			s = last
		}
	}

	s = strings.ReplaceAll(s, "_", "-")
	return s
}

// SplitProviderInput splits comma-separated provider names and normalizes each non-empty entry.
func SplitProviderInput(raw string) []string {
	var result []string
	for _, part := range strings.Split(raw, ",") {
		norm := NormalizeProviderName(part)
		if norm != "" && !containsString(result, norm) {
			result = append(result, norm)
		}
	}
	return result
}

func containsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

// LoadConfig reads the persistent media configuration from cache.
// If the file does not exist yet, an empty Config is returned without error.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := cache.Load(MediaConfigFile, &cfg); err != nil {
		return &Config{AllowedProviders: []string{}}, nil
	}
	// Normalize loaded entries
	var normalized []string
	for _, p := range cfg.AllowedProviders {
		norm := NormalizeProviderName(p)
		if norm != "" && !containsString(normalized, norm) {
			normalized = append(normalized, norm)
		}
	}
	cfg.AllowedProviders = normalized
	return &cfg, nil
}

// SaveConfig persists the media configuration to cache.
func SaveConfig(cfg *Config) error {
	if cfg.AllowedProviders == nil {
		cfg.AllowedProviders = []string{}
	}
	return cache.Save(MediaConfigFile, cfg)
}

// ToggleProvider flips whether a provider is allowed to play during Salat break slots.
// Returns true if the provider is now allowed, or false if it is now disallowed.
func ToggleProvider(provider string) (bool, []string, error) {
	norm := NormalizeProviderName(provider)
	if norm == "" {
		cfg, _ := LoadConfig()
		return false, cfg.AllowedProviders, nil
	}

	cfg, _ := LoadConfig()
	var updated []string
	found := false
	for _, p := range cfg.AllowedProviders {
		if p == norm {
			found = true
			continue
		}
		updated = append(updated, p)
	}

	allowed := false
	if !found {
		updated = append(updated, norm)
		allowed = true
	}

	cfg.AllowedProviders = updated
	if err := SaveConfig(cfg); err != nil {
		return allowed, updated, err
	}
	return allowed, updated, nil
}

// AllowProviders adds one or more providers to the Salat slot allowlist.
func AllowProviders(providers ...string) ([]string, error) {
	cfg, _ := LoadConfig()
	for _, raw := range providers {
		for _, norm := range SplitProviderInput(raw) {
			if !containsString(cfg.AllowedProviders, norm) {
				cfg.AllowedProviders = append(cfg.AllowedProviders, norm)
			}
		}
	}
	if err := SaveConfig(cfg); err != nil {
		return cfg.AllowedProviders, err
	}
	return cfg.AllowedProviders, nil
}

// DisallowProviders removes one or more providers from the Salat slot allowlist.
func DisallowProviders(providers ...string) ([]string, error) {
	cfg, _ := LoadConfig()
	var toRemove []string
	for _, raw := range providers {
		for _, norm := range SplitProviderInput(raw) {
			toRemove = append(toRemove, norm)
		}
	}

	var updated []string
	for _, p := range cfg.AllowedProviders {
		if !containsString(toRemove, p) {
			updated = append(updated, p)
		}
	}
	cfg.AllowedProviders = updated
	if err := SaveConfig(cfg); err != nil {
		return cfg.AllowedProviders, err
	}
	return cfg.AllowedProviders, nil
}
