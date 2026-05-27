#!/usr/bin/env bash
# install.sh — served from raw.githubusercontent.com
# Install HI CLI tool

set -e

REPO="Oridjinnn/hi"
BIN_NAME="hi"
INSTALL_DIR="/usr/local/bin"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
esac

echo "HI — Installing for $OS/$ARCH..."

# Get latest release
VERSION=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')

if [ -z "$VERSION" ]; then
  echo "Could not determine latest version. Trying to use main branch..."
  VERSION="latest"
fi

echo "Version: v$VERSION"

# Download binary
URL="https://github.com/$REPO/releases/download/v$VERSION/${BIN_NAME}_${OS}_${ARCH}"
echo "Downloading from: $URL"

if command -v curl &> /dev/null; then
  curl -fsSL "$URL" -o "/tmp/$BIN_NAME"
elif command -v wget &> /dev/null; then
  wget -q "$URL" -O "/tmp/$BIN_NAME"
else
  echo "Error: need curl or wget to download"
  exit 1
fi

chmod +x "/tmp/$BIN_NAME"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "/tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  echo "Need sudo to install to $INSTALL_DIR"
  sudo mv "/tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
fi

echo ""
echo "✓ hi v$VERSION installed to $INSTALL_DIR/$BIN_NAME"
echo ""
echo "  Next steps:"
echo "  1. Run 'hi auth login' to authenticate with GitHub"
echo "  2. Run 'hi' to launch the signal feed"
echo "  3. Run 'hi --help' for all commands"
echo ""
echo "  Need help? https://github.com/$REPO"