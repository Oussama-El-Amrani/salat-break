#!/bin/bash

# Salat Break Setup Script
# Automates building and installing the Salat Break service.

APP_DIR="/data/dev/self-hosted/salat-break"
SERVICE_NAME="salat-break"
SERVICE_FILE="$HOME/.config/systemd/user/$SERVICE_NAME.service"

echo "=== Setting up Salat Break ==="

# 1. Check/Install dependencies
if ! command -v dbus-send &> /dev/null; then
    echo "dbus-send not found. Attempting to install..."
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        case "$ID" in
            ubuntu|debian)
                sudo apt-get update && sudo apt-get install -y dbus libnotify-bin
                ;;
            fedora|centos|rhel)
                sudo dnf install -y dbus libnotify
                ;;
            arch)
                sudo pacman -S --noconfirm dbus libnotify
                ;;
            *)
                echo "Warning: Unknown distribution: $ID. Please install dbus-x11 or dbus manually if needed."
                ;;
        esac
    else
        echo "Warning: Could not detect distribution. Please install dbus manually if needed."
    fi
fi

# 2. Inform about media player control
echo "Checking for media players (Spotify, Chrome, etc.)..."
MEDIA_PLAYER_FOUND=0

# Check for Spotify (various methods)
if command -v spotify &> /dev/null || (command -v snap &> /dev/null && snap list spotify &> /dev/null) || (command -v flatpak &> /dev/null && flatpak list --columns=application | grep -q "com.spotify.Client") || (command -v dpkg &> /dev/null && dpkg -l spotify-client &> /dev/null); then
    echo "Found Spotify."
    MEDIA_PLAYER_FOUND=1
fi

# Check for Chromium/Chrome
if command -v chromium &> /dev/null || command -v chromium-browser &> /dev/null || command -v google-chrome &> /dev/null; then
    echo "Found a browser (Chromium/Chrome) supported for YouTube pause."
    MEDIA_PLAYER_FOUND=1
fi

if [ $MEDIA_PLAYER_FOUND -eq 0 ]; then
    echo "Note: No common media players (Spotify/Chrome) were detected."
    echo "The service will still run and will automatically control any MPRIS-compliant player you install later."
else
    echo "Common media players detected successfully."
fi

# 3. Create directory if not exists
mkdir -p "$APP_DIR"
cd "$APP_DIR" || exit

# 4. Check for main.go
if [ ! -f "main.go" ]; then
    echo "Error: main.go not found in $APP_DIR. Please ensure the code is present."
    exit 1
fi

# 5. Initialize Go module if needed
if [ ! -f "go.mod" ]; then
    echo "Initializing Go module..."
    go mod init github.com/oussama_ib0/salat-break
fi

# 6. Build the application
echo "Building the application..."
go build -o "$SERVICE_NAME" main.go
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# 7. Create systemd user service directory
mkdir -p "$HOME/.config/systemd/user/"

# 8. Generate systemd service file
echo "Generating systemd service file..."
UID_VAL=$(id -u)
cat <<EOF > "$SERVICE_FILE"
[Unit]
Description=Salat Break Service
After=network.target

[Service]
ExecStart=$APP_DIR/$SERVICE_NAME
Restart=always
Environment=DISPLAY=:0
Environment=DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$UID_VAL/bus

[Install]
WantedBy=default.target
EOF

# 9. Reload systemd and start service
echo "Reloading systemd and starting service..."
systemctl --user daemon-reload
systemctl --user enable "$SERVICE_NAME"
systemctl --user restart "$SERVICE_NAME"

echo "=== Setup Complete! ==="
echo "Check status with: systemctl --user status $SERVICE_NAME"
echo "Check logs with: journalctl --user -u $SERVICE_NAME -f"
