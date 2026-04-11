package location

import (
	"fmt"
	"time"

	"github.com/godbus/dbus/v5"
)

// tryGeoClue2 attempts to get location from the GeoClue2 D-Bus service.
// GeoClue2 uses WiFi triangulation, GPS, and other system-level sources
// to provide accurate location — similar to how mobile devices work.
func tryGeoClue2() (*Location, error) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot connect to system bus: %w", err)
	}
	defer conn.Close()

	manager := conn.Object("org.freedesktop.GeoClue2", "/org/freedesktop/GeoClue2/Manager")

	// Request a client from the GeoClue2 Manager
	var clientPath dbus.ObjectPath
	err = manager.Call("org.freedesktop.GeoClue2.Manager.GetClient", 0).Store(&clientPath)
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot get client: %w", err)
	}

	client := conn.Object("org.freedesktop.GeoClue2", clientPath)

	// Set the desktop ID (required by GeoClue2 for authorization)
	err = client.SetProperty("org.freedesktop.GeoClue2.Client.DesktopId", dbus.MakeVariant("salat-break"))
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot set desktop id: %w", err)
	}

	// Set requested accuracy level: EXACT (8) for best possible accuracy
	err = client.SetProperty("org.freedesktop.GeoClue2.Client.RequestedAccuracyLevel", dbus.MakeVariant(uint32(8)))
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot set accuracy level: %w", err)
	}

	// Subscribe to LocationUpdated signal
	sigChan := make(chan *dbus.Signal, 1)
	conn.Signal(sigChan)

	matchRule := fmt.Sprintf(
		"type='signal',sender='org.freedesktop.GeoClue2',path='%s',interface='org.freedesktop.GeoClue2.Client',member='LocationUpdated'",
		clientPath,
	)
	err = conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule).Err
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot add signal match: %w", err)
	}

	// Start the client
	err = client.Call("org.freedesktop.GeoClue2.Client.Start", 0).Err
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot start client: %w", err)
	}

	// Wait for a location update signal (timeout after 10 seconds)
	var locationPath dbus.ObjectPath
	select {
	case sig := <-sigChan:
		if len(sig.Body) >= 2 {
			locationPath = sig.Body[1].(dbus.ObjectPath)
		}
	case <-time.After(10 * time.Second):
		// Stop the client before returning
		_ = client.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err
		return nil, fmt.Errorf("geoclue2: timed out waiting for location")
	}

	// Read coordinates from the location object
	locObj := conn.Object("org.freedesktop.GeoClue2", locationPath)

	latVariant, err := locObj.GetProperty("org.freedesktop.GeoClue2.Location.Latitude")
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot get latitude: %w", err)
	}
	lonVariant, err := locObj.GetProperty("org.freedesktop.GeoClue2.Location.Longitude")
	if err != nil {
		return nil, fmt.Errorf("geoclue2: cannot get longitude: %w", err)
	}
	accVariant, err := locObj.GetProperty("org.freedesktop.GeoClue2.Location.Accuracy")
	if err != nil {
		// Silent
	}

	lat, ok := latVariant.Value().(float64)
	if !ok {
		return nil, fmt.Errorf("geoclue2: latitude is not float64")
	}
	lon, ok := lonVariant.Value().(float64)
	if !ok {
		return nil, fmt.Errorf("geoclue2: longitude is not float64")
	}

	accuracy := 0.0
	if accVariant.Value() != nil {
		if a, ok := accVariant.Value().(float64); ok {
			accuracy = a
		}
	}

	// Stop the client
	_ = client.Call("org.freedesktop.GeoClue2.Client.Stop", 0).Err

	return &Location{
		Lat:      lat,
		Lon:      lon,
		Accuracy: accuracy,
		Source:   "geoclue2",
	}, nil
}
