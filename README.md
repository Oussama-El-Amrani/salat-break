# 🕌 Salat Break

[![Release](https://img.shields.io/github/v/release/Oussama-El-Amrani/salat-break?display_name=tag)](https://github.com/Oussama-El-Amrani/salat-break/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://go.dev/)

**Salat Break** is a lightweight, intelligent background service for Linux that automatically pauses your music players (Spotify, Rhythmbox, etc.) during prayer times. It stays out of your way until it's time for prayer, ensuring you never miss a Salat while listening to media.

---

## ✨ Features

- 🌍 **Auto-Location**: Detects your city and timezone automatically using secure IP-based geolocation.
- 🕒 **Precision Timing**: Fetches daily prayer times from the Aladhan API with daily local caching.
- 🎵 **Universal Control**: Pauses **Spotify, Rhythmbox, Clementine, etc.** via the MPRIS DBus interface.
- 🧠 **Smart Detection**: Recognizes browser content (YouTube, etc.) to avoid pausing tutorials while still sending notifications.
- 🔔 **Desktop Alerts**: Sends a native Linux notification before the prayer starts.
- 🕊️ **Zero Overhead**: Written in Go, it uses minimal system resources and runs as a `systemd` user service.
- 🔌 **Offline First**: Works without an internet connection by falling back to cached location and timings.
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

- **Linux** (Tested on Ubuntu, Debian, Fedora, Arch)
- `dbus-send` (provided by `dbus-x11` or `dbus`)
- `notify-send` (provided by `libnotify-bin`)
- **Spotify** or any MPRIS-compatible media player.

---

## ⚙️ Usage & Configuration

Salat Break runs as a `systemd` user service, but you can also use the CLI to manage your configuration.

### CLI Flags

| Flag | Description |
| :--- | :--- |
| `--show-timings` | Display today's prayer times and exit. |
| `--city "Name"` | Manually override the auto-detected city. |
| `--country "Name"`| Manually override the auto-detected country. |
| `--method ID` | Set a specific [calculation method](https://aladhan.com/calculation-methods) (e.g., 21 for Morocco, 3 for MWL). |

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

1. **Geolocation**: Uses `ipwhois.app` to resolve your location.
2. **Caching**: Timings and location are stored in `~/.cache/salat-break/` to prevent redundant API calls.
3. **Observation Loop**: Every 30 seconds, the app checks if the current time falls within the window (**3 minutes before** to **3 minutes after** the prayer).
4. **Media Interception**: Sends a `Pause` signal to all music players. Non-music media (like videos) triggers a notification only.
5. **Persistence**: Manual configuration overrides are stored in `~/.cache/salat-break/location_override.json`.

---

## 🤝 Contributing

Contributions are welcome! Whether it's a bug report, a feature request, or a pull request, we appreciate your help in making Salat Break better for everyone.

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
