#!/usr/bin/env bash
set -euo pipefail

REPO="zsoftly/zcp-cli"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY_NAME="zcp"

info() { printf "\033[0;36m  %s\033[0m\n" "$1"; }
ok()   { printf "\033[0;32m  [OK] %s\033[0m\n" "$1"; }
err()  { printf "\033[0;31m  [ERROR] %s\033[0m\n" "$1"; exit 1; }

echo ""
echo "  ZCP CLI Installer"
echo "  -----------------"
echo ""

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux"  ;;
  mingw*|msys*|cygwin*) err "Use PowerShell installer on Windows" ;;
  *) err "Unsupported OS: $OS" ;;
esac

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) err "Unsupported arch: $ARCH" ;;
esac

ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}"
DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${ASSET_NAME}"

info "Downloading ${ASSET_NAME}..."
TMP_FILE=$(mktemp)
trap 'rm -f "$TMP_FILE"' EXIT

if command -v curl &>/dev/null; then
  curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"
elif command -v wget &>/dev/null; then
  wget -q "$DOWNLOAD_URL" -O "$TMP_FILE"
else
  err "curl or wget required"
fi

chmod +x "$TMP_FILE"

if [ ! -w "$INSTALL_DIR" ]; then
  info "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
else
  mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
fi

ok "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
echo ""
info "Run: ${BINARY_NAME} version"
echo ""
