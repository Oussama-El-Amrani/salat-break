#!/bin/bash

# Salat Break One-Line Installer
# Usage: curl -sSL https://raw.githubusercontent.com/Oussama-El-Amrani/salat-break/main/install.sh | bash

set -e

# Colors
BOLD="$(tput bold)"
GREEN="$(tput setaf 2)"
YELLOW="$(tput setaf 3)"
CYAN="$(tput setaf 6)"
RED="$(tput setaf 1)"
RESET="$(tput sgr0)"

REPO="Oussama-El-Amrani/salat-break"
SETUP_SCRIPT="setup.sh"

# Detect OS and Architecture
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "${RED}Unsupported architecture: $ARCH${RESET}"; exit 1 ;;
esac

BINARY_NAME="salat-break-$OS-$ARCH"

echo "${BOLD}${CYAN}=== Salat Break Installer ===${RESET}"

# 1. Create a temporary directory
TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT
cd "$TMP_DIR"

# 2. Get the latest release tag
echo "Fetching latest version information..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "${YELLOW}Warning: Could not find latest release tag. Using main branch.${RESET}"
    LATEST_TAG="main"
fi

# 3. Download setup.sh
echo "Downloading setup script..."
curl -sSL -O "https://raw.githubusercontent.com/$REPO/main/$SETUP_SCRIPT"

# 4. Handle Binary or Source
echo "Attempting to download pre-built binary ($LATEST_TAG)..."
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$BINARY_NAME"

if curl -sSL -f -O "$DOWNLOAD_URL"; then
    mv "$BINARY_NAME" "salat-break"
    echo "${GREEN}Binary downloaded successfully.${RESET}"
else
    echo "${YELLOW}Pre-built binary not found or unsupported architecture.${RESET}"
    if command -v go &> /dev/null && command -v git &> /dev/null; then
        echo "Git and Go detected. Falling back to source build..."
        git clone --depth 1 "https://github.com/$REPO.git" .
    else
        echo "${RED}Error: Could not download binary and 'go'/'git' are not available for a source build.${RESET}"
        echo "Please visit https://github.com/$REPO/releases to download manually."
        exit 1
    fi
fi

# 5. Run the setup script
chmod +x "$SETUP_SCRIPT"
./"$SETUP_SCRIPT"
