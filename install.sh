#!/bin/bash

# Salat Break One-Line Installer
# Usage: curl -sSL https://raw.githubusercontent.com/Oussama-El-Amrani/salat-break/main/install.sh | bash

set -e

REPO="Oussama-El-Amrani/salat-break"
BINARY_NAME="salat-break-linux-amd64"
SETUP_SCRIPT="setup.sh"

echo "=== Salat Break Installer ==="

# 1. Create a temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

# 2. Get the latest release tag
echo "Fetching latest release version..."
LATEST_TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Could not find latest release tag. Falling back to main branch for scripts."
    LATEST_TAG="main"
fi

# 3. Download setup.sh and binary
echo "Downloading $SETUP_SCRIPT..."
curl -sSL -O "https://raw.githubusercontent.com/$REPO/main/$SETUP_SCRIPT"

echo "Downloading binary ($LATEST_TAG)..."
# We try to get it from releases, if latest tag failed we might need to fallback or error
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$BINARY_NAME"
# If it's the first time and there's no release yet, this will fail. 
# In that case, we tell the user to build from source.

if ! curl -sSL -f -O "$DOWNLOAD_URL"; then
    echo "Warning: Could not download pre-built binary for $LATEST_TAG."
    echo "I will try to build from source if 'go' is installed."
else
    # Rename specifically for setup.sh to find it
    mv "$BINARY_NAME" "salat-break"
fi

# 4. Run the setup script
chmod +x "$SETUP_SCRIPT"
./"$SETUP_SCRIPT"

# 5. Cleanup
cd - > /dev/null
rm -rf "$TMP_DIR"
