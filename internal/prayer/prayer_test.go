package prayer

import (
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/Oussama-El-Amrani/salat-break/internal/cache"
	"github.com/Oussama-El-Amrani/salat-break/internal/location"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestFormatTimings(t *testing.T) {
	pt := &PrayerTimes{}
	pt.Data.Timings = map[string]string{
		"Fajr":    "05:00",
		"Dhuhr":   "12:30",
		"Asr":     "16:00",
		"Maghrib": "19:00",
		"Isha":    "20:30",
		"Sunset":  "18:50", // Should be ignored
	}

	expected := "Fajr: 05:00 | Dhuhr: 12:30 | Asr: 16:00 | Maghrib: 19:00 | Isha: 20:30"
	assert.Equal(t, expected, pt.FormatTimings())
}

func TestGetDefaultMethod(t *testing.T) {
	tests := []struct {
		countryCode string
		expected    int
	}{
		{"MA", 21},
		{"SA", 4},
		{"EG", 5},
		{"US", 0}, // Default
		{"fr", 12}, // Case insensitivity
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.countryCode, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetDefaultMethod(tt.countryCode))
		})
	}
}

func TestService_BuildCacheKey(t *testing.T) {
	s := &Service{Method: 21}
	loc := &location.Location{
		Lat: 33.5731,
		Lon: -7.5898,
	}
	date := "01-01-2026"

	key1 := s.buildCacheKey(loc, date)
	
	// Slight variation in coordinates should yield SAME key (due to rounding)
	loc2 := &location.Location{
		Lat: 33.5741,
		Lon: -7.5888,
	}
	key2 := s.buildCacheKey(loc2, date)

	assert.Equal(t, key1, key2, "Cache keys should be stable for minor GPS drift")
	assert.Contains(t, key1, "33.57")
	assert.Contains(t, key1, "-7.59") // Rounded
}

func TestGetPrayerTimes(t *testing.T) {
	tmpDir, _ := os.MkdirTemp("", "salat-break-prayer-test-*")
	defer os.RemoveAll(tmpDir)
	cache.BaseDir = tmpDir

	mockClient := new(MockHTTPClient)
	s := NewService(mockClient)

	loc := &location.Location{
		City:    "Casablanca",
		Country: "Morocco",
		Lat:     33.5731,
		Lon:     -7.5898,
		Source:  "test",
	}

	// Mock for monthly fetch (first thing GetPrayerTimes does)
	// We return a 404 to force fallback to single day fetch for simplicity of this test
	// OR we can mock the full calendar response. Let's mock a single day failure.
	mockClient.On("Get", mock.MatchedBy(func(url string) bool {
		return strings.Contains(url, "calendar")
	})).Return(&http.Response{
		StatusCode: http.StatusNotFound,
		Body:       io.NopCloser(strings.NewReader("{}")),
	}, nil)

	// Mock for today's timings (fallback)
	todayJSON := `{"data": {"timings": {"Fajr": "05:00", "Dhuhr": "12:30", "Asr": "16:00", "Maghrib": "19:00", "Isha": "20:30"}}}`
	mockClient.On("Get", mock.MatchedBy(func(url string) bool {
		return strings.Contains(url, "timings")
	})).Return(&http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(todayJSON)),
	}, nil)

	pt, err := s.GetPrayerTimes(loc)
	assert.NoError(t, err)
	assert.NotNil(t, pt)
	assert.Equal(t, "05:00", pt.Data.Timings["Fajr"])

	mockClient.AssertExpectations(t)
}
