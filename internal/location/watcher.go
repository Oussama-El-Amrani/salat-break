package location

import (
	"log"
	"time"
)

// LocationResolver defines the interface for location resolution services.
type LocationResolver interface {
	GetLocation() (*Location, error)
	CurrentFingerprint() string
}

// Watcher manages location refresh logic based on environmental changes.
type Watcher struct {
	svc             LocationResolver
	refreshInterval time.Duration

	// State for testing/internal use
	lastFingerprint string
	lastLocation    *Location
	lastRefresh     time.Time
}

// NewWatcher creates a new location watcher.
func NewWatcher(svc LocationResolver) *Watcher {
	return &Watcher{
		svc:             svc,
		refreshInterval: 6 * time.Hour, // Force refresh every 6 hours
	}
}

// GetLocation returns the current location. It only performs a full resolution
// if the environment (Network Fingerprint) has changed or if enough time has passed.
func (w *Watcher) GetLocation() (*Location, error) {
	fingerprint := w.svc.CurrentFingerprint()
	
	now := time.Now()
	needsRefresh := false

	// Case 1: First run
	if w.lastLocation == nil {
		needsRefresh = true
		logVerbose("Watcher: Initial location fetch...")
	} else if fingerprint != w.lastFingerprint && fingerprint != "unknown" {
		// Case 2: Network changed (and we have a valid fingerprint)
		needsRefresh = true
		log.Printf("Watcher: Network environment changed (%s -> %s). Refreshing location...", 
			w.lastFingerprint, fingerprint)
	} else if now.Sub(w.lastRefresh) > w.refreshInterval {
		// Case 3: Forced periodic refresh
		needsRefresh = true
		logVerbose("Watcher: Periodic refresh triggered (%v elapsed)", w.refreshInterval)
	}

	if !needsRefresh {
		return w.lastLocation, nil
	}

	// Perform actual resolution
	loc, err := w.svc.GetLocation()
	if err != nil {
		// If resolution fails, keep using last location if available
		if w.lastLocation != nil {
			log.Printf("Watcher: Failed to refresh location (%v), using cached location.", err)
			return w.lastLocation, nil
		}
		return nil, err
	}

	// Update state
	w.lastFingerprint = fingerprint
	w.lastLocation = loc
	w.lastRefresh = now

	return loc, nil
}
