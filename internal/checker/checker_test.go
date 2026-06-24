package checker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
)

type MockMediaManager struct {
	mock.Mock
}

func (m *MockMediaManager) PauseAllPlayers() {
	m.Called()
}

func (m *MockMediaManager) PlayAllPlayers() {
	m.Called()
}

func (m *MockMediaManager) EnforceSilence() {
	m.Called()
}

type MockNotifier struct {
	mock.Mock
}

func (m *MockNotifier) SendNotification(title, message string) {
	m.Called(title, message)
}

func TestCheckAndPause(t *testing.T) {
	timings := map[string]string{
		"Dhuhr": "12:00",
	}

	t.Run("Entering prayer window", func(t *testing.T) {
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)

		// 11:58:00 is within the window (12:00 +/- 3 mins)
		now := time.Date(2026, 3, 31, 11, 58, 0, 0, time.Local)

		mockNotify.On("SendNotification", "Salat Break", "It is time for Dhuhr (12:00).").Once()
		mockMedia.On("PauseAllPlayers").Once()

		s.CheckAndPause(timings, now)

		mockNotify.AssertExpectations(t)
		mockMedia.AssertExpectations(t)
	})

	t.Run("Exiting prayer window", func(t *testing.T) {
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)

		// Set initial state as if we are in a break
		s.inPrayerBreak = true
		s.lastHandledPrayer = "Dhuhr"

		// 12:04:00 is outside the window
		now := time.Date(2026, 3, 31, 12, 4, 0, 0, time.Local)

		mockNotify.On("SendNotification", "Salat Break Ended", "The prayer window has ended. Media playback resumed.").Once()
		mockMedia.On("PlayAllPlayers").Once()

		s.CheckAndPause(timings, now)

		mockNotify.AssertExpectations(t)
		mockMedia.AssertExpectations(t)
	})

	t.Run("Staying within prayer window - default strict mode", func(t *testing.T) {
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)
		s.AllowPlayback = false // Default

		// Set initial state as already in break
		s.inPrayerBreak = true
		s.lastHandledPrayer = "Dhuhr"

		// 12:01:00 is within the window
		now := time.Date(2026, 3, 31, 12, 1, 0, 0, time.Local)

		// Should call EnforceSilence but NOT notify again
		mockMedia.On("EnforceSilence").Once()

		s.CheckAndPause(timings, now)

		mockNotify.AssertNotCalled(t, "SendNotification")
		mockMedia.AssertExpectations(t)
	})

	t.Run("Staying within prayer window - allow playback mode", func(t *testing.T) {
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)
		s.AllowPlayback = true

		// Set initial state as already in break
		s.inPrayerBreak = true
		s.lastHandledPrayer = "Dhuhr"

		// 12:01:00 is within the window
		now := time.Date(2026, 3, 31, 12, 1, 0, 0, time.Local)

		// Should NOT call notify or pause again (either PauseAllPlayers or EnforceSilence)
		s.CheckAndPause(timings, now)

		mockNotify.AssertNotCalled(t, "SendNotification")
		mockMedia.AssertNotCalled(t, "PauseAllPlayers")
		mockMedia.AssertNotCalled(t, "EnforceSilence")
	})

	t.Run("Handling timezone suffix in timings", func(t *testing.T) {
		suffixTimings := map[string]string{
			"Dhuhr": "12:00 (+01)",
		}
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)

		// 12:00:00 is within the window
		now := time.Date(2026, 3, 31, 12, 0, 0, 0, time.Local)

		mockNotify.On("SendNotification", "Salat Break", "It is time for Dhuhr (12:00 (+01)).").Once()
		mockMedia.On("PauseAllPlayers").Once()

		s.CheckAndPause(suffixTimings, now)

		mockNotify.AssertExpectations(t)
		mockMedia.AssertExpectations(t)
	})

	t.Run("Exiting prayer window with auto-play disabled", func(t *testing.T) {
		mockMedia := new(MockMediaManager)
		mockNotify := new(MockNotifier)
		s := NewService(mockMedia, mockNotify)
		s.AutoPlay = false

		// Set initial state as if we are in a break
		s.inPrayerBreak = true
		s.lastHandledPrayer = "Dhuhr"

		// 12:04:00 is outside the window
		now := time.Date(2026, 3, 31, 12, 4, 0, 0, time.Local)

		mockNotify.On("SendNotification", "Salat Break Ended", "The prayer window has ended.").Once()
		// PlayAllPlayers should NOT be called

		s.CheckAndPause(timings, now)

		mockNotify.AssertExpectations(t)
		mockMedia.AssertNotCalled(t, "PlayAllPlayers")
	})
}
