package location

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLocationResolver is a mock implementation of LocationResolver.
type MockLocationResolver struct {
	mock.Mock
}

func (m *MockLocationResolver) GetLocation() (*Location, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Location), args.Error(1)
}

func (m *MockLocationResolver) CurrentFingerprint() string {
	args := m.Called()
	return args.String(0)
}

func TestWatcher_GetLocation(t *testing.T) {
	t.Run("Initial fetch", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)

		expectedLoc := &Location{City: "Casablanca"}
		mockSvc.On("CurrentFingerprint").Return("ssid:home")
		mockSvc.On("GetLocation").Return(expectedLoc, nil)

		loc, err := watcher.GetLocation()
		assert.NoError(t, err)
		assert.Equal(t, expectedLoc, loc)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Network change triggers refresh", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)

		loc1 := &Location{City: "Casablanca"}
		loc2 := &Location{City: "Fès"}

		// First call
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()
		_, _ = watcher.GetLocation()

		// Second call with different SSID
		mockSvc.On("CurrentFingerprint").Return("ssid:work").Once()
		mockSvc.On("GetLocation").Return(loc2, nil).Once()

		loc, err := watcher.GetLocation()
		assert.NoError(t, err)
		assert.Equal(t, loc2, loc)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Same network uses cache", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)

		loc1 := &Location{City: "Casablanca"}

		// First call
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()
		_, _ = watcher.GetLocation()

		// Second call with same SSID - should NOT call GetLocation again
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()

		loc, err := watcher.GetLocation()
		assert.NoError(t, err)
		assert.Equal(t, loc1, loc)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Offline (unknown) does NOT trigger refresh", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)

		loc1 := &Location{City: "Casablanca"}

		// First call
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()
		_, _ = watcher.GetLocation()

		// Second call while offline (unknown) - should NOT call GetLocation
		mockSvc.On("CurrentFingerprint").Return("unknown").Once()

		loc, err := watcher.GetLocation()
		assert.NoError(t, err)
		assert.Equal(t, loc1, loc)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Periodic refresh", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)
		watcher.refreshInterval = 1 * time.Millisecond

		loc1 := &Location{City: "Casablanca"}

		// First call
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()
		_, _ = watcher.GetLocation()

		time.Sleep(2 * time.Millisecond)

		// Second call after interval - should call GetLocation even if SSID same
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()

		_, _ = watcher.GetLocation()
		mockSvc.AssertExpectations(t)
	})

	t.Run("Fallback to cache on error", func(t *testing.T) {
		mockSvc := new(MockLocationResolver)
		watcher := NewWatcher(mockSvc)

		loc1 := &Location{City: "Casablanca"}

		// First call
		mockSvc.On("CurrentFingerprint").Return("ssid:home").Once()
		mockSvc.On("GetLocation").Return(loc1, nil).Once()
		_, _ = watcher.GetLocation()

		// Second call with different SSID but GetLocation fails
		mockSvc.On("CurrentFingerprint").Return("ssid:work").Once()
		mockSvc.On("GetLocation").Return(nil, errors.New("API down")).Once()

		loc, err := watcher.GetLocation()
		assert.NoError(t, err) // No error because we fell back to cache
		assert.Equal(t, loc1, loc)
		mockSvc.AssertExpectations(t)
	})
}
