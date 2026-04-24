package media

import (
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

func (c *Controller) IsMusic(player string, title, artist string) bool {
	player = strings.ToLower(player)
	title = strings.ToLower(title)
	artist = strings.ToLower(artist)

	musicPlayers := []string{"spotify", "youtube_music", "rhythmbox", "clementine", "mpd", "audacious", "music", "itunes", "music.app"}
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
