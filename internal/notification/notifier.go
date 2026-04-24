package notification

import (
	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
)

type Service struct {
	timeout            int
	clearDelay         int
	lastNotificationID uint32
}

func NewService(timeout, clearDelay int) *Service {
	s := &Service{
		timeout:    timeout,
		clearDelay: clearDelay,
	}
	// Initial load of last ID
	_ = cache.Load("last_notification_id.json", &s.lastNotificationID)
	return s
}
