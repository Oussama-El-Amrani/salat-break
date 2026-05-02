# 🕌 Salat Break

[![Release](https://img.shields.io/github/v/release/Oussama-El-Amrani/salat-break?display_name=tag)](https://github.com/Oussama-El-Amrani/salat-break/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

**Salat Break** is a lightweight, intelligent background service for Linux and macOS that automatically pauses your music players (Spotify, Music.app, Rhythmbox, etc.) during prayer times. It stays out of your way until it's time for prayer, ensuring you never miss a Salat while listening to media.

---

## ✨ Features

- 🌍 **Smart Location Detection**: Automatically detects when you move (e.g., traveling) by monitoring your WiFi environment and updates prayer times immediately.
- 🕒 **Precision Geolocation**: Uses WiFi Triangulation and IP Consensus to find your exact coordinates for maximum accuracy.
- 🕒 **Coordinate-Based Timing**: Fetches daily prayer times using exact latitude and longitude for higher precision compared to simple city names.
- 🎵 **Universal Control**: Pauses **Spotify, Music.app, Rhythmbox, Clementine, etc.** via MPRIS (Linux) or AppleScript (macOS).
- 🧠 **Smart Detection**: Recognizes browser content (YouTube, etc.) to avoid pausing tutorials while still sending notifications.
- 🔔 **Desktop Alerts**: Sends native Linux/macOS notifications before the prayer starts.
- 🕊️ **Zero Overhead**: Written in Go, it uses minimal system resources and runs as a background service (`systemd` or `launchd`).
- 🔌 **Offline First**: Monthly pre-fetching ensures you have the entire month's prayer times cached locally, staying fully functional even during extended internet outages.
- ⚙️ **Configurable**: Manually override your city, country, or prayer calculation method via CLI.

---

## 🚀 Installation

To install Salat Break with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/Oussama-El-Amrani/salat-break/main/install.sh | bash
```

Alternatively, you can clone the repository and run the setup script:

```bash
git clone https://github.com/Oussama-El-Amrani/salat-break.git
cd salat-break
./setup.sh
```

The installer will auto-detect your location and start the background service immediately.

---

## 🛠️ Requirements

- **Linux** (Tested on Ubuntu, Debian, Fedora, Arch) or **macOS** (⚠️ Not fully tested - if you encounter any issues, please open an issue)
- **Linux Dependencies**: `dbus-send` (provided by `dbus-x11`), `notify-send` (provided by `libnotify-bin`)
- **macOS Dependencies**: None (uses built-in AppleScript and `airport`)
- **Spotify**, **Music.app**, or any MPRIS-compatible media player.

---

## ⚙️ Usage & Configuration

Salat Break runs as a `systemd` user service, but you can also use the CLI to manage your configuration.

### CLI Flags

| Flag | Short | Description |
| :--- | :--- | :--- |
| `--show-timings` | `-t` | Display today's prayer times and exit. |
| `--show-location`| `-l` | Display your current calculated location (lat/lon, city, source) and exit. |
| `--verbose` | `-v` | Show detailed internal logs for location resolution and scanning. |
| `--lat` / `--lon` | | Manually set your exact coordinates for maximum precision. |
| `--reset-location`| | Clear all manual location overrides and return to automatic detection. |
| `--city "Name"` | | Manually override the auto-detected city. |
| `--country "Name"`| | Manually override the auto-detected country. |
| `--method ID` | | Set a specific [calculation method](https://aladhan.com/calculation-methods) (e.g., 21 for Morocco). |
| `--auto-play` | | Automatically resume media playback after the prayer break ends (default `true`). |
| `--notification-timeout`| | Timeout for notifications in ms (0 to hide the popup). |
| `--version` | | Display the current installed version. |
| `update` | | **Subcommand**: Update salat-break to the latest version automatically. |

### Testing & Debugging

| Flag | Description |
| :--- | :--- |
| `--test-pause` | Trigger a manual music pause test. |
| `--test-play` | Trigger a manual music resume test. |
| `--test-notify`| Send a test notification. |

> [!TIP]
> Changing the city, country, or method via the CLI will automatically save your preference and restart the background service to apply the changes.

### Service Management

| Action | Command |
| :--- | :--- |
| **Check Status** | `systemctl --user status salat-break` |
| **Restart Service** | `systemctl --user restart salat-break` |
| **Stop Service** | `systemctl --user stop salat-break` |
| **View Logs** | `journalctl --user -u salat-break -f` |
| **Update Tool** | `salat-break update` |

---

## 🏗️ Architecture

1. **Geolocation**: Uses a prioritised multi-source strategy (see [GEOLOCATION_LOGIC.md](./GEOLOCATION_LOGIC.md)):
    - **WiFi Triangulation**: Scans nearby APs via `nmcli` and resolves coordinates via the BeaconDB API.
    - **IP Consensus**: Queries multiple IP providers simultaneously (ipinfo.io, ip-api.com, ipwhois.app) to find a median consensus point.
2. **Reverse Geocoding**: Converts coordinates to local city/country names via OpenStreetMap.
3. **Monthly Pre-fetching**: On the first run or at the start of a month, the app fetches the entire month's calendar at once. Each day is cached individually, providing weeks of offline resilience.
4. **Observation Loop**: 
    - **Prayer Check**: Every 30 seconds, the app checks if it's time for prayer.
    - **Environment Monitoring**: Monitors your WiFi SSID or Public IP. If your environment changes, it automatically refreshes your location and updates prayer times.
4. **Media Interception**: Sends a `Pause` signal to all music players. Non-music media (like videos) triggers a notification only.
    - **Auto-Resume**: By default, the service will automatically resume any media it paused once the prayer window (3 minutes after the prayer time) ends. You can disable this with `--auto-play=false`.
5. **Persistence**: Manual configuration overrides are stored in `~/.cache/salat-break/location_override.json`.

---

## 🤝 Contributing

Contributions are what make the open source community such an amazing place to learn, inspire, and create. Any contributions you make are **greatly appreciated**.

*   **Found a bug?** [Open an issue](https://github.com/Oussama-El-Amrani/salat-break/issues/new?labels=bug) and describe the problem.
*   **Want a new feature?** [Open an issue](https://github.com/Oussama-El-Amrani/salat-break/issues/new?labels=enhancement) to discuss your idea.
*   **Ready to code?** Fork the repo and submit a Pull Request!

### Development Workflow

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

---

## 📜 License

Distributed under the **MIT License**. See `LICENSE` for more information.

---

<p align="center">Made with ❤️</p>
