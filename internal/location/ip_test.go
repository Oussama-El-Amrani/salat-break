package location

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHaversineDistance(t *testing.T) {
	// Casablanca (33.5731, -7.5898) to Rabat (34.0209, -6.8416)
	// Approx 87km
	d := haversineDistance(33.5731, -7.5898, 34.0209, -6.8416)
	assert.InDelta(t, 87.0, d, 2.0)

	// Same point
	assert.Equal(t, 0.0, haversineDistance(33.5731, -7.5898, 33.5731, -7.5898))
}

func TestMedianLocation(t *testing.T) {
	locs := []*Location{
		{Lat: 33.5, Lon: -7.5},
		{Lat: 34.0, Lon: -6.8},
		{Lat: 33.0, Lon: -8.0}, // Outlier
	}

	median := medianLocation(locs)
	// Sorted lats: 33.0, 33.5, 34.0 -> Median: 33.5
	// Sorted lons: -8.0, -7.5, -6.8 -> Median: -7.5
	assert.Equal(t, 33.5, median.Lat)
	assert.Equal(t, -7.5, median.Lon)
}

func TestFindClosest(t *testing.T) {
	target := &Location{Lat: 33.6, Lon: -7.4}
	locs := []*Location{
		{Lat: 33.5, Lon: -7.5, City: "Casablanca"},
		{Lat: 34.0, Lon: -6.8, City: "Rabat"},
	}

	closest := findClosest(target, locs)
	assert.Equal(t, "Casablanca", closest.City)
}
