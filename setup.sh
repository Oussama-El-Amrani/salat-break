#!/bin/bash

# Salat Break Setup Script
# Automates building and installing the Salat Break service on Linux.

set -e

# Colors for better UI
BOLD="$(tput bold)"
GREEN="$(tput setaf 2)"
YELLOW="$(tput setaf 3)"
CYAN="$(tput setaf 6)"
RED="$(tput setaf 1)"
RESET="$(tput sgr0)"

APP_NAME="salat-break"
INSTALL_DIR="$HOME/.local/bin"
SERVICE_NAME="salat-break"
SERVICE_FILE="$HOME/.config/systemd/user/$SERVICE_NAME.service"

echo "${BOLD}${CYAN}=== Salat Break Setup ===${RESET}"

# Navigation to script directory if not already there
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# macOS Delegation
if [[ "$OSTYPE" == "darwin"* ]]; then
    echo "macOS detected. Switching to macOS setup..."
    chmod +x ./setup_macos.sh
    ./setup_macos.sh "$@"
    exit $?
fi

# 0. Uninstall Option
if [[ "$1" == "--uninstall" ]]; then
    echo "${YELLOW}Uninstalling $APP_NAME...${RESET}"
    systemctl --user stop "$SERVICE_NAME" &> /dev/null || true
    systemctl --user disable "$SERVICE_NAME" &> /dev/null || true
    rm -f "$SERVICE_FILE"
    rm -f "$INSTALL_DIR/$APP_NAME"
    echo "${GREEN}Salat Break has been removed.${RESET}"
    exit 0
fi

# 1. Dependency check
check_dependency() {
    local cmd=$1
    local pkg=$2
    if ! command -v "$cmd" &> /dev/null; then
        echo "${YELLOW}Dependency '$cmd' not found. Trying to install $pkg...${RESET}"
        if [ -f /etc/os-release ]; then
            . /etc/os-release
            case "$ID" in
                ubuntu|debian|pop|raspbian)
                    sudo apt-get update && sudo apt-get install -y "$pkg"
                    ;;
                fedora|centos|rhel)
                    sudo dnf install -y "$pkg"
                    ;;
                arch|manjaro)
                    sudo pacman -S --noconfirm "$pkg"
                    ;;
                *)
                    echo "${RED}Warning: Unknown distribution. Please install $pkg manually.${RESET}"
                    ;;
            esac
        else
            echo "${RED}Warning: Could not detect OS. Please install $pkg manually.${RESET}"
        fi
    fi
}

check_dependency "dbus-send" "dbus-x11"
check_dependency "notify-send" "libnotify-bin"
check_dependency "curl" "curl"

# 2. Version Detection and Update Check
INSTALLED_VER=""
if [ -f "$INSTALL_DIR/$APP_NAME" ]; then
    INSTALLED_VER=$("$INSTALL_DIR/$APP_NAME" --version 2>/dev/null | awk '{print $NF}')
    if [ -n "$INSTALLED_VER" ]; then
        echo "Currently installed version: ${BOLD}$INSTALLED_VER${RESET}"
    fi
fi

echo "Checking for the latest version on GitHub..."
REPO="Oussama-El-Amrani/salat-break"
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -n "$LATEST_TAG" ]; then
    echo "Latest version available: ${BOLD}$LATEST_TAG${RESET}"
    if [ "$INSTALLED_VER" == "$LATEST_TAG" ]; then
        echo "${GREEN}Salat Break is already at the latest version ($INSTALLED_VER).${RESET}"
        read -p "Do you want to re-install? [y/N] " response
        if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            echo "Setup cancelled. You're already up to date!"
            exit 0
        fi
    fi
else
    echo "${YELLOW}Warning: Could not fetch latest version from GitHub. Proceeding...${RESET}"
fi

# 3. Build or Use Existing Binary
if [ -f "./$APP_NAME" ]; then
    echo "Using existing binary found in current directory..."
elif command -v go &> /dev/null; then
    echo "Go detected. ${BOLD}Building from source...${RESET}"
    VERSION_STR=$(git describe --tags --always --dirty 2>/dev/null || echo "$LATEST_TAG")
    CGO_ENABLED=0 go build -ldflags "-X main.Version=$VERSION_STR" -o "$APP_NAME" ./cmd/salat-break
else
    echo "${RED}Error: Binary not found and 'go' is not installed.${RESET}"
    echo "Please download the binary manually or install Go 1.21+."
    exit 1
fi

# 3. Create necessary directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$HOME/.config/systemd/user/"

# 4. Copy binary to local bin
echo "Installing binary to $INSTALL_DIR..."
systemctl --user stop "$SERVICE_NAME" &> /dev/null || true
pkill -f "$APP_NAME" &> /dev/null || true
sleep 1
install -m 755 "$APP_NAME" "$INSTALL_DIR/$APP_NAME"

# 5. Generate/Update systemd service file
echo "Generating systemd service file..."
UID_VAL=$(id -u)

cat <<SVC > "$SERVICE_FILE"
[Unit]
Description=Salat Break Service
After=network.target
PartOf=graphical-session.target

[Service]
ExecStart=$INSTALL_DIR/$APP_NAME
Restart=always
RestartSec=10
Environment=DISPLAY=:0
Environment=DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/$UID_VAL/bus

[Install]
WantedBy=default.target
SVC

# 6. Location Configuration
echo "${BOLD}${CYAN}=== Location Configuration ===${RESET}"
echo "Verifying location detection..."

# Use the binary's own detection logic (Internal WiFi + IP Consensus)
# This is much more accurate than the previous curl-based method.
if "$INSTALL_DIR/$APP_NAME" --show-location; then
    echo ""
    echo "${GREEN}Location successfully detected!${RESET}"
    echo "Salat Break will use your coordinates to calculate precise prayer times."
    echo ""
    echo "If this is incorrect, you can manually override it later with:"
    echo "  $APP_NAME --city \"Casablanca\""
    echo ""
else
    echo "${YELLOW}Warning: Automatic location detection failed.${RESET}"
    echo "You might need to set your location manually:"
    echo "  $APP_NAME --city \"YourCity\""
fi

# 7. Reload and enable service
echo "Reloading systemd and starting service..."
systemctl --user daemon-reload
systemctl --user enable "$SERVICE_NAME"
systemctl --user restart "$SERVICE_NAME"

# 8. Final Check
echo ""
if command -v "$APP_NAME" &> /dev/null; then
    echo "${BOLD}${GREEN}✔ Salat Break is installed and running!${RESET}"
else
    echo "${BOLD}${YELLOW}⚠ Installation completed, but '$INSTALL_DIR' might not be in your PATH.${RESET}"
    echo "Add this to your .bashrc or .zshrc:"
    echo "  export PATH=\$PATH:\$HOME/.local/bin"
fi

echo ""
echo "Check status: ${CYAN}systemctl --user status $SERVICE_NAME${RESET}"
echo "View logs:   ${CYAN}journalctl --user -u $SERVICE_NAME -f${RESET}"
echo ""
echo "Enjoy your peaceful prayer breaks!"
