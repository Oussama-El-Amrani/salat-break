package location

import (
	"log"
	"time"
)

// Watcher manages location refresh logic based on environmental changes.
type Watcher struct {
	svc             *Service
	refreshInterval time.Duration
}

// NewWatcher creates a new location watcher.
func NewWatcher(svc *Service) *Watcher {
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
	if w.svc.lastLocation == nil {
		needsRefresh = true
		logVerbose("Watcher: Initial location fetch...")
	} else if fingerprint != w.svc.lastFingerprint && fingerprint != "unknown" {
		// Case 2: Network changed (and we have a valid fingerprint)
		needsRefresh = true
		log.Printf("Watcher: Network environment changed (%s -> %s). Refreshing location...", 
			w.svc.lastFingerprint, fingerprint)
	} else if now.Sub(w.svc.lastRefresh) > w.refreshInterval {
		// Case 3: Forced periodic refresh
		needsRefresh = true
		logVerbose("Watcher: Periodic refresh triggered (%v elapsed)", w.refreshInterval)
	}

	if !needsRefresh {
		return w.svc.lastLocation, nil
	}

	// Perform actual resolution
	loc, err := w.svc.GetLocation()
	if err != nil {
		// If resolution fails, keep using last location if available
		if w.svc.lastLocation != nil {
			log.Printf("Watcher: Failed to refresh location (%v), using cached location.", err)
			return w.svc.lastLocation, nil
		}
		return nil, err
	}

	// Update state
	w.svc.lastFingerprint = fingerprint
	w.svc.lastLocation = loc
	w.svc.lastRefresh = now

	return loc, nil
}
