# Salat Spotify Break

A lightweight Go application for Linux that automatically pauses Spotify around prayer times (Salat). It runs in the background as a `systemd` user service and detects your location automatically to fetch accurate prayer times.

## Features
- **Auto-Location**: Detects your city and timezone using IP-based geolocation.
- **Prayer Time Sync**: Fetches daily prayer times from the Aladhan API (Standard method).
- **Spotify Integration**: Pauses Spotify 2 minutes before the Adhan and keeps it paused for 5 minutes total (until 3 minutes after the prayer time).
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
1.  **Geolocation**: On startup, the app calls `ip-api.com` to get your current city and country.
2.  **Prayer Times**: Uses the city/country to fetch today's prayer times from `api.aladhan.com`.
3.  **Monitoring**: Checks every 30 seconds if the current time falls within the pause window (`[PrayerTime - 2m, PrayerTime + 3m]`).
4.  **DBus Command**: If in the window, it sends a `Pause` signal to Spotify via the MPRIS DBus interface.

## License
MIT
