# Salat Break

A lightweight Go application for Linux that automatically pauses media players around prayer times (Salat). It runs in the background as a `systemd` user service and detects your location automatically to fetch accurate prayer times.

## Features
- **Auto-Location**: Detects your city and timezone using secure IP-based geolocation (`ipwhois.app`).
- **Prayer Time Sync**: Fetches daily prayer times from the Aladhan API over HTTPS.
- **Local Caching**: Stores location and prayer times in `~/.cache/salat-break/` to ensure the app works **offline** and to prevent redundant API calls.
- **Security Hardened**: Encrypted communications (HTTPS), input sanitization, and DBus service validation.
- **Universal Media Control**: Automatically pauses **all active music** (Spotify, Rhythmbox, etc.) 2 minutes before prayer and resumes after.
- **Intelligent Media Detection**: Recognizes browser content (YouTube, etc.) and avoids pausing video tutorials while still sending a desktop notification.
- **Desktop Notifications**: Alerts you 2 minutes before each prayer so you can prepare.
- **Background Service**: Managed by `systemd` to ensure it starts automatically with your user session.
- **Easy Setup**: Includes a `setup.sh` script to handle dependencies, building, and service activation.

## Prerequisites
- **Go**: 1.18 or higher.
- **dbus-send**: To communicate with Spotify (the setup script will attempt to install this for you).
- **Spotify**: Should be installed (supports apt, snap, or flatpak).
- **Linux**: Designed to work with common Linux desktop environments.

## Installation

1.  **Clone the repository**:
    ```bash
    git clone https://github.com/Oussama-El-Amrani/salat-spotify-break.git
    cd salat-spotify-break
    ```

2.  **Run the setup script**:
    ```bash
    chmod +x setup.sh
    ./setup.sh
    ```
    This will:
    - Check for dependencies (`dbus-send`, Spotify).
    - Build the Go application.
    - Create and enable the `systemd` user service.

## Usage

You can manage the background service using standard `systemctl` commands:

- **Check status**:
  ```bash
  systemctl --user status salat-break
  ```
- **View logs**:
  ```bash
  journalctl --user -u salat-break -f
  ```
- **Restart**:
  ```bash
  systemctl --user restart salat-break
  ```
- **Stop**:
  ```bash
  systemctl --user stop salat-break
  ```

### Manual Testing
You can manually test the Spotify control functionality using the following flags:
- **Pause**: `./salat-break -test-pause`
- **Play**: `./salat-break -test-play`

## How It Works
1.  **Geolocation**: On startup, the app calls a secure API (`https://ipwhois.app/json/`) to get your current location. This is cached in `last_location.json`.
2.  **Prayer Times**: Uses your location to fetch today's prayer times from `https://api.aladhan.com`. These are cached locally (`prayer_times_*.json`) for each day.
3.  **Offline Support**: If the internet is down, the app automatically falls back to your last known location and previously cached timings.
4.  **Monitoring**: Checks every 30 seconds if the current time falls within the pause window (`[PrayerTime - 2m, PrayerTime + 3m]`).
5.  **Media Control**: If in the window, it sends a `Pause` signal to all music players via the MPRIS DBus interface. Browsers/Video players are notified but not paused.

## License
MIT
