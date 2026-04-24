#!/bin/bash

# Salat Break macOS Setup Script
# Automates building and installing the Salat Break service on macOS.

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
AGENT_NAME="com.oussama.salat-break"
AGENT_FILE="$HOME/Library/LaunchAgents/$AGENT_NAME.plist"

echo "${BOLD}${CYAN}=== Salat Break macOS Setup ===${RESET}"

# 0. Uninstall Option
if [[ "$1" == "--uninstall" ]]; then
    echo "${YELLOW}Uninstalling $APP_NAME...${RESET}"
    launchctl bootout "gui/$(id -u)/$AGENT_NAME" &> /dev/null || true
    rm -f "$AGENT_FILE"
    rm -f "$INSTALL_DIR/$APP_NAME"
    echo "${GREEN}Salat Break has been removed.${RESET}"
    exit 0
fi

# 1. Dependency check
# macOS has osascript and airport built-in. curl is also standard.
# We just need to check if Go is installed if we need to build from source.

# 2. Build or Use Existing Binary
if [ -f "./$APP_NAME" ]; then
    echo "Using existing binary found in current directory..."
elif command -v go &> /dev/null; then
    echo "Go detected. ${BOLD}Building from source...${RESET}"
    
    # Try to get version from Git, fallback to GitHub API, then hardcoded
    VERSION_STR=$(git describe --tags --always --dirty 2>/dev/null || true)
    if [ -z "$VERSION_STR" ]; then
        REPO="Oussama-El-Amrani/salat-break"
        VERSION_STR=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || echo "v0.1.0")
    fi
    
    CGO_ENABLED=0 go build -ldflags "-X main.Version=$VERSION_STR" -o "$APP_NAME" ./cmd/salat-break
else
    echo "${RED}Error: Binary not found and 'go' is not installed.${RESET}"
    echo "Please download the macOS binary manually or install Go."
    exit 1
fi

# 3. Create necessary directories
mkdir -p "$INSTALL_DIR"
mkdir -p "$HOME/Library/LaunchAgents"

# 4. Copy binary to local bin
echo "Installing binary to $INSTALL_DIR..."
# Stop agent if running
launchctl bootout "gui/$(id -u)/$AGENT_NAME" &> /dev/null || true
pkill -f "$APP_NAME" &> /dev/null || true
sleep 1
cp "$APP_NAME" "$INSTALL_DIR/$APP_NAME"
chmod +x "$INSTALL_DIR/$APP_NAME"

# 5. Generate/Update launchd agent file
echo "Generating launchd agent file..."

cat <<PLIST > "$AGENT_FILE"
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>$AGENT_NAME</string>
    <key>ProgramArguments</key>
    <array>
        <string>$INSTALL_DIR/$APP_NAME</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/$APP_NAME.stdout.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/$APP_NAME.stderr.log</string>
</dict>
</plist>
PLIST

# 6. Load and start agent
echo "Loading and starting background service..."
launchctl bootstrap "gui/$(id -u)" "$AGENT_FILE"

# 7. Final Check
echo ""
if command -v "$APP_NAME" &> /dev/null; then
    echo "${BOLD}${GREEN}✔ Salat Break is installed and running!${RESET}"
else
    echo "${BOLD}${YELLOW}⚠ Installation completed, but '$INSTALL_DIR' might not be in your PATH.${RESET}"
    echo "Add this to your .bash_profile or .zshrc:"
    echo "  export PATH=\$PATH:\$HOME/.local/bin"
fi

echo ""
echo "Check logs: ${CYAN}tail -f /tmp/$APP_NAME.stderr.log${RESET}"
echo ""
echo "Enjoy your peaceful prayer breaks!"
