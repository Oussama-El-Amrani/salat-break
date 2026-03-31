#!/bin/bash

# Salat Break Setup Script
# Automates building and installing the Salat Break service.

APP_DIR="/data/dev/self-hosted/salat-break"
SERVICE_NAME="salat-break"
SERVICE_FILE="$HOME/.config/systemd/user/$SERVICE_NAME.service"

echo "=== Setting up Salat Break (Modular) ==="

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
            *)
                echo "Warning: Unknown distribution. Please install dbus manually."
                ;;
        esac
    fi
fi

# 2. Build the application
echo "Building the application..."
cd "$APP_DIR" || exit
go build -o "$SERVICE_NAME" ./cmd/salat-break
if [ $? -ne 0 ]; then
    echo "Build failed!"
    exit 1
fi

# 3. Create systemd user service directory
mkdir -p "$HOME/.config/systemd/user/"

# 4. Generate systemd service file
echo "Generating systemd service file..."
UID_VAL=$(id -u)
cat <<SVC > "$SERVICE_FILE"
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
SVC

# 5. Reload systemd and start service
echo "Reloading systemd and starting service..."
systemctl --user daemon-reload
systemctl --user enable "$SERVICE_NAME"
systemctl --user restart "$SERVICE_NAME"

echo "=== Setup Complete! ==="
echo "Check status with: systemctl --user status $SERVICE_NAME"
