#!/bin/bash

# Salat Break Setup Script
# Automates building and installing the Salat Break service on Linux.

set -e

APP_NAME="salat-break"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_NAME="salat-break"
SERVICE_FILE="$HOME/.config/systemd/user/$SERVICE_NAME.service"

echo "=== Setting up $APP_NAME ==="

# 1. Dependency check
check_dependency() {
    local cmd=$1
    local pkg=$2
    if ! command -v "$cmd" &> /dev/null; then
        echo "$cmd not found. Attempting to install $pkg..."
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            case "$ID" in
                ubuntu|debian|pop)
                    sudo apt-get update && sudo apt-get install -y "$pkg"
                    ;;
                fedora|centos|rhel)
                    sudo dnf install -y "$pkg"
                    ;;
                arch|manjaro)
                    sudo pacman -S --noconfirm "$pkg"
                    ;;
                *)
                    echo "Warning: Unknown distribution. Please install $pkg manually."
                    ;;
            esac
        fi
    fi
}

check_dependency "dbus-send" "dbus-x11"
check_dependency "notify-send" "libnotify-bin"

# 2. Version Detection and Update Check
INSTALLED_VER=""
if [ -f "$INSTALL_DIR/$APP_NAME" ]; then
    INSTALLED_VER=$("$INSTALL_DIR/$APP_NAME" --version 2>/dev/null | awk '{print $NF}')
    if [ -n "$INSTALLED_VER" ]; then
        echo "Currently installed version: $INSTALLED_VER"
    fi
fi

echo "Checking for the latest version on GitHub..."
REPO="Oussama-El-Amrani/salat-break"
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -n "$LATEST_TAG" ]; then
    echo "Latest version available: $LATEST_TAG"
    if [ "$INSTALLED_VER" == "$LATEST_TAG" ]; then
        echo "Salat Break is already at the latest version ($INSTALLED_VER)."
        read -p "Do you want to forced reinstall? [y/N] " response
        if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            echo "Setup cancelled. You're already up to date!"
            exit 0
        fi
    fi
else
    echo "Warning: Could not fetch latest version from GitHub. Proceeding with standard setup..."
fi

# 3. Build or Use Existing Binary
if [ -f "./$APP_NAME" ]; then
    echo "Using existing binary in current directory..."
elif command -v go &> /dev/null; then
    echo "Go detected. Building from source with dynamic version..."
    # Get current version details for injection
    VERSION_STR=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
    CGO_ENABLED=0 go build -ldflags "-X main.Version=$VERSION_STR" -o "$APP_NAME" ./cmd/salat-break
else
    echo "Error: Binary not found and 'go' is not installed."
    echo "Please download the binary or install Go 1.21+."
    exit 1
fi

# 3. Create necessary directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$HOME/.config/systemd/user/"

# 4. Copy binary to local bin and ensure it's executable
echo "Installing binary..."
# Stop service if it's already running
systemctl --user stop "$SERVICE_NAME" &> /dev/null || true
pkill -f "$APP_NAME" &> /dev/null || true
sleep 1
# Use install command which handles unlinking busy binaries better
mkdir -p "$INSTALL_DIR"
install -m 755 "$APP_NAME" "$INSTALL_DIR/$APP_NAME"
# Cleanup any backup if it exists
rm -f "$INSTALL_DIR/$APP_NAME.bak" &> /dev/null || true

# 5. Generate/Update systemd service file
echo "Generating systemd service file..."
UID_VAL=$(id -u)

cat <<SVC > "$SERVICE_FILE"
[Unit]
Description=Salat Break Service
After=network.target

[Service]
ExecStart=$INSTALL_DIR/$APP_NAME
Restart=always
Environment=DISPLAY=:0
Environment=DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$UID_VAL/bus

[Install]
WantedBy=default.target
SVC

# 6. Location Configuration
echo "=== Location Configuration ==="
LOC_JSON=$(curl -s https://ipwhois.app/json/ || echo "")
if [ -n "$LOC_JSON" ] && [[ "$LOC_JSON" == *'"success":true'* ]]; then
    DETECTED_CITY=$(echo "$LOC_JSON" | grep -o '"city":"[^"]*' | cut -d'"' -f4)
    DETECTED_COUNTRY=$(echo "$LOC_JSON" | grep -o '"country":"[^"]*' | cut -d'"' -f4)
    echo "Auto-detected location: $DETECTED_CITY, $DETECTED_COUNTRY"
    echo "Warning: Location auto-detection depends on your ISP and may not always be accurate."
    echo "If incorrect, you can manually set your city using:"
    echo "  $APP_NAME --city \"Casablanca\""
    echo ""
    
    # Run the binary with --show-timings to verify and display timings
    "$INSTALL_DIR/$APP_NAME" --show-timings
else
    echo "Could not auto-detect location. You can configure it manually using:"
    echo "  $APP_NAME --city \"YourCity\""
fi

# 7. Reload and enable service
echo "Reloading systemd and starting service..."
systemctl --user daemon-reload
systemctl --user enable "$SERVICE_NAME"
systemctl --user restart "$SERVICE_NAME"

echo "=== Setup Complete! ==="
echo "Salat Break is now running in the background."
echo "Check status: systemctl --user status $SERVICE_NAME"
echo "View logs: journalctl --user -u $SERVICE_NAME -f"
