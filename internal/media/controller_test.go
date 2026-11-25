package media

import (
	"os"
	"testing"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopNotifier struct{}

func (n *noopNotifier) SendNotification(title, message string) {}

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"VLC", "vlc"},
		{"  vlc  ", "vlc"},
		{"org.mpris.MediaPlayer2.vlc", "vlc"},
		{"org.mpris.MediaPlayer2.vlc.instance12345", "vlc"},
		{"org.mpris.MediaPlayer2.org.videolan.VLC", "vlc"},
		{"org.mpris.MediaPlayer2.io.bassi.Amberol", "amberol"},
		{"org.mpris.MediaPlayer2.com.spotify.Client", "spotify"},
		{"Spotify-Web-Chrome", "spotify"},
		{"Web-youtube-music-Chrome", "youtube-music"},
		{"Music.app", "music"},
		{"youtube_music", "youtube-music"},
		{"", ""},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.expected, NormalizeProviderName(tc.input), "input: %s", tc.input)
	}
}

func TestIsMusic_AnyProviderAndSmartBrowser(t *testing.T) {
	ctrl := NewController(&noopNotifier{})

	// Standalone desktop/MPRIS players should ALWAYS be supported regardless of track title
	standalonePlayers := []string{
		"org.mpris.MediaPlayer2.vlc",
		"org.mpris.MediaPlayer2.vlc.instance999",
		"org.mpris.MediaPlayer2.mpv",
		"org.mpris.MediaPlayer2.spotify",
		"org.mpris.MediaPlayer2.rhythmbox",
		"org.mpris.MediaPlayer2.io.bassi.Amberol",
		"org.mpris.MediaPlayer2.strawberry",
		"org.mpris.MediaPlayer2.org.kde.elisa",
		"VLC",
		"VOX",
		"IINA",
		"Music",
	}

	for _, p := range standalonePlayers {
		meta := map[string]string{"title": "Untitled Track 01"}
		assert.True(t, ctrl.IsMusic(p, meta), "standalone player %s should be supported", p)
	}

	// Browser playing web music providers should be controlled
	assert.True(t, ctrl.IsMusic("org.mpris.MediaPlayer2.chromium.instance123", map[string]string{
		"title":  "Track Name",
		"artUrl": "https://i.scdn.co/image/123",
	}))
	assert.True(t, ctrl.IsMusic("org.mpris.MediaPlayer2.firefox.instance_1_42", map[string]string{
		"title": "My Track",
		"url":   "https://music.youtube.com/watch?v=abc",
	}))
	assert.True(t, ctrl.IsMusic("org.mpris.MediaPlayer2.firefox.instance_1_42", map[string]string{
		"title": "Ambient Stream",
		"url":   "https://soundcloud.com/artist/track",
	}))
	assert.True(t, ctrl.IsMusic("org.mpris.MediaPlayer2.brave.instance1", map[string]string{
		"title": "Track",
		"url":   "https://www.deezer.com/track/123",
	}))

	// Browser playing non-music tutorial video should NOT be paused
	assert.False(t, ctrl.IsMusic("org.mpris.MediaPlayer2.firefox.instance_1_42", map[string]string{
		"title": "Go Concurrency Patterns Tutorial",
		"url":   "https://www.youtube.com/watch?v=xyz",
	}))
}

func TestIsProviderAllowed(t *testing.T) {
	ctrl := NewController(&noopNotifier{})
	ctrl.SetAllowedProviders([]string{"VLC", "youtube_music"})

	assert.True(t, ctrl.IsProviderAllowed("org.mpris.MediaPlayer2.vlc", map[string]string{"title": "Recitation.mp3"}))
	assert.True(t, ctrl.IsProviderAllowed("org.mpris.MediaPlayer2.vlc.instance42", nil))
	assert.True(t, ctrl.IsProviderAllowed("VLC", nil))
	assert.True(t, ctrl.IsProviderAllowed("org.mpris.MediaPlayer2.chromium.instance10", map[string]string{
		"url": "https://music.youtube.com/watch?v=123",
	}))

	assert.False(t, ctrl.IsProviderAllowed("org.mpris.MediaPlayer2.spotify", map[string]string{"title": "Song"}))
	assert.False(t, ctrl.IsProviderAllowed("org.mpris.MediaPlayer2.mpv", map[string]string{"title": "Clip"}))
}

func TestToggleAllowDisallowPersistence(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "salat-break-media-cfg-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	prevBase := cache.BaseDir
	cache.BaseDir = tmpDir
	defer func() { cache.BaseDir = prevBase }()

	// Initially empty
	cfg, err := LoadConfig()
	require.NoError(t, err)
	assert.Empty(t, cfg.AllowedProviders)

	// Toggle VLC ON
	allowed, list, err := ToggleProvider("VLC")
	require.NoError(t, err)
	assert.True(t, allowed)
	assert.Equal(t, []string{"vlc"}, list)

	// Allow mpv and spotify
	list, err = AllowProviders("mpv, org.mpris.MediaPlayer2.spotify")
	require.NoError(t, err)
	assert.Equal(t, []string{"vlc", "mpv", "spotify"}, list)

	// Toggle VLC OFF
	allowed, list, err = ToggleProvider("org.mpris.MediaPlayer2.vlc")
	require.NoError(t, err)
	assert.False(t, allowed)
	assert.Equal(t, []string{"mpv", "spotify"}, list)

	// Disallow mpv
	list, err = DisallowProviders("mpv")
	require.NoError(t, err)
	assert.Equal(t, []string{"spotify"}, list)
}
