#!/usr/bin/env bash
set -e

# RIP Quick Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/nxhung1610/proxy-ipv6-generator-cli/main/install.sh | bash

REPO="nxhung1610/proxy-ipv6-generator-cli"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY_NAME="rip"

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Linux*)  echo "linux" ;;
    Darwin*) echo "darwin" ;;
    MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
    *) echo "unsupported" ;;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    armv7l)        echo "arm" ;;
    *)
      echo "unsupported"
      return 1
      ;;
  esac
}

# Detect package format
detect_ext() {
  local os=$1
  case $os in
    windows) echo ".exe" ;;
    *)       echo "" ;;
  esac
}

# Print colored messages
info() { echo -e "\033[1;34m[INFO]\033[0m $1"; }
warn() { echo -e "\033[1;33m[WARN]\033[0m $1"; }
error() { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }
success() { echo -e "\033[1;32m[OK]\033[0m $1"; }

# Get latest release version
get_latest_version() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
    grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install() {
  local os=$1
  local arch=$2
  local ext
  ext=$(detect_ext "$os")

  if [[ "$os" == "unsupported" || "$arch" == "unsupported" ]]; then
    error "Unsupported platform: $(uname -s) $(uname -m)"
    error "Supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64"
    exit 1
  fi

  local version
  version=$(get_latest_version)
  if [[ -z "$version" ]]; then
    error "Failed to fetch latest release version"
    exit 1
  fi

  local filename="rip-${os}-${arch}${ext}"
  local url="https://github.com/${REPO}/releases/download/${version}/${filename}"

  info "Installing RIP ${version} for ${os}/${arch}..."
  info "Downloading: ${filename}"

  local tmp_dir
  tmp_dir=$(mktemp -d)
  local archive="${tmp_dir}/${filename}"

  # Download with retry
  local retries=3
  for i in $(seq 1 $retries); do
    if curl -fSL --connect-timeout 15 --max-time 120 \
      -o "$archive" "$url" 2>/dev/null; then
      break
    fi
    warn "Download attempt $i failed, retrying..."
    if [[ $i -eq $retries ]]; then
      rm -rf "$tmp_dir"
      error "Failed to download after ${retries} attempts"
      exit 1
    fi
  done

  # Create install directory
  mkdir -p "$INSTALL_DIR"

  # Install binary
  local dest="${INSTALL_DIR}/${BINARY_NAME}${ext}"
  mv "$archive" "$dest"
  chmod +x "$dest"

  # Cleanup
  rm -rf "$tmp_dir"

  success "Installed ${BINARY_NAME} to ${dest}"

  # Verify
  if "$dest" version >/dev/null 2>&1; then
    success "Verification passed"
  else
    warn "Binary may need executable permissions: chmod +x ${dest}"
  fi

  # Check if INSTALL_DIR is in PATH
  if [[ ":$PATH:" == *":${INSTALL_DIR}:"* ]]; then
    info "✓ ${INSTALL_DIR} is in your PATH"
  else
    warn "${INSTALL_DIR} is not in your PATH."
    info "Add this to your shell config:"
    if [[ -n "$ZSH_VERSION" ]]; then
      info "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.zshrc"
    else
      info "  echo 'export PATH=\"\$HOME/.local/bin:\$PATH\"' >> ~/.bashrc"
    fi
  fi

  info "Run 'rip --help' to get started!"
}

# Install 3proxy
install_3proxy() {
  local os
  local arch
  os=$1
  arch=$2

  info "Installing 3proxy..."

  local tmp_dir
  tmp_dir=$(mktemp -d)
  cd "$tmp_dir"

  local file
  local url

  case "$os" in
    linux)
      file="3proxy-${arch}.tar.gz"
      url="https://github.com/3proxy/3proxy/releases/download/0.9.6/${file}"
      if curl -fsSL -o "$file" "$url" 2>/dev/null && tar -xzf "$file" 2>/dev/null; then
        mv 3proxy "${INSTALL_DIR}/3proxy" && chmod +x "${INSTALL_DIR}/3proxy" && \
          success "3proxy installed to ${INSTALL_DIR}/3proxy"
      else
        warn "Could not download 3proxy ${arch} binary; install via apt/yum or build from source"
      fi
      ;;
    darwin)
      # No official macOS binary; suggest brew
      warn "No official 3proxy macOS binary. Install via brew:"
      info "  brew install 3proxy"
      ;;
    windows)
      warn "3proxy Windows installation not yet supported by this script"
      ;;
  esac

  rm -rf "$tmp_dir"
  cd - >/dev/null
}

# Main
main() {
  echo ""
  echo "  RIP Installer — IPv6 Proxy Pool Manager"
  echo "  https://github.com/${REPO}"
  echo ""

  local os
  local arch
  os=$(detect_os)
  arch=$(detect_arch)

  info "Detected: ${os}/${arch}"

  install "$os" "$arch"

  # Optionally install 3proxy
  if [[ "${INSTALL_3PROXY:-false}" == "true" ]]; then
    install_3proxy "$os" "$arch"
  fi
}

main "$@"
