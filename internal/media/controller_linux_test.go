//go:build linux

package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeSilence_AutoPausesReplayedVLC(t *testing.T) {
	tmpBin, err := os.MkdirTemp("", "salat-break-rt-mocks-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpBin)

	dbusLogPath := filepath.Join(tmpBin, "dbus.log")

	// Create mock dbus-monitor that simulates VLC emitting PlaybackStatus = "Playing"
	dbusMonitorScript := `#!/bin/bash
echo 'signal time=1700000000 sender=:1.42 -> destination=(null destination) serial=10 path=/org/mpris/MediaPlayer2; interface=org.freedesktop.DBus.Properties; member=PropertiesChanged'
echo '   string "org.mpris.MediaPlayer2.Player"'
echo '   variant       string "Playing"'
sleep 5
`
	err = os.WriteFile(filepath.Join(tmpBin, "dbus-monitor"), []byte(dbusMonitorScript), 0755)
	require.NoError(t, err)

	// Create mock dbus-send reporting VLC as Playing and logging Pause calls
	dbusSendScript := `#!/bin/bash
if [[ "$*" == *"org.freedesktop.DBus.ListNames"* ]]; then
    echo '   string "org.mpris.MediaPlayer2.vlc"'
    exit 0
fi
if [[ "$*" == *"PlaybackStatus"* ]]; then
    echo '   variant       string "Playing"'
    exit 0
fi
if [[ "$*" == *"Metadata"* ]]; then
    echo '   variant       array ['
    echo '      dict entry('
    echo '         string "xesam:title"'
    echo '         variant string "Replay Test.mp3"'
    echo '      )'
    echo '   ]'
    exit 0
fi
if [[ "$*" == *"org.mpris.MediaPlayer2.Player.Pause"* ]]; then
    echo "PAUSED:$*" >> "` + dbusLogPath + `"
    exit 0
fi
exit 0
`
	err = os.WriteFile(filepath.Join(tmpBin, "dbus-send"), []byte(dbusSendScript), 0755)
	require.NoError(t, err)

	prevPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpBin+":"+prevPath)
	defer os.Setenv("PATH", prevPath)

	ctrl := NewController(&noopNotifier{})
	ctrl.StartRealtimeSilence()
	defer ctrl.StopRealtimeSilence()

	require.Eventually(t, func() bool {
		data, err := os.ReadFile(dbusLogPath)
		return err == nil && len(data) > 0
	}, 1*time.Second, 25*time.Millisecond, "VLC should be auto-paused immediately upon Playing signal")

	data, err := os.ReadFile(dbusLogPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "org.mpris.MediaPlayer2.vlc")
}
