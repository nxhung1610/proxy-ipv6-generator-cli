#!/usr/bin/env bash
set -e

# RIP Quick Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/nxhung1610/proxy-ipv6-generator-cli/main/install.sh | bash

REPO="nxhung1610/proxy-ipv6-generator-cli"
INSTALL_DIR="${INSTALL_DIR:-${HOME}/.local/bin}"
BINARY_NAME="rip"

# Detect OS (WSL aware)
detect_os() {
  case "$(uname -s)" in
    Linux*)
      if grep -qi 'microsoft\|wsl' /proc/version 2>/dev/null; then
        echo "wsl"
      else
        echo "linux"
      fi
      ;;
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

# Download and install rip
install() {
  local os=$1
  local arch=$2

  if [[ "$os" == "unsupported" || "$arch" == "unsupported" ]]; then
    error "Unsupported platform: $(uname -s) $(uname -m)"
    error "Supported: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64, windows/amd64, wsl/amd64, wsl/arm64"
    exit 1
  fi

  # WSL maps to linux binary
  local release_os=$os
  if [[ "$os" == "wsl" ]]; then
    release_os="linux"
  fi

  local version
  version=$(get_latest_version)
  if [[ -z "$version" ]]; then
    error "Failed to fetch latest release version"
    exit 1
  fi

  local ext=""
  local url_ext=""
  if [[ "$os" == "windows" ]]; then
    ext=".exe"
    url_ext=".zip"
  else
    url_ext=".tar.gz"
  fi

  local filename="rip-${release_os}-${arch}${url_ext}"
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

  # Extract if needed
  if [[ "$url_ext" == ".tar.gz" ]]; then
    tar -xzf "$archive" -C "$tmp_dir"
    rm -f "$archive"
    archive=$(find "$tmp_dir" -type f -name "rip*" ! -name "*.tar.gz" | head -1)
  elif [[ "$url_ext" == ".zip" ]]; then
    unzip -o "$archive" -d "$tmp_dir"
    rm -f "$archive"
    archive=$(find "$tmp_dir" -type f -name "rip*.exe" | head -1)
  fi

  if [[ -z "$archive" || ! -f "$archive" ]]; then
    error "Failed to extract binary from archive"
    rm -rf "$tmp_dir"
    exit 1
  fi

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
    info "PATH check: ${INSTALL_DIR} is in your PATH"
  else
    warn "${INSTALL_DIR} is not in your PATH."
    local rc=""
    if [[ -n "$ZSH_VERSION" ]]; then
      rc="~/.zshrc"
    elif [[ -f "$HOME/.bashrc" ]]; then
      rc="~/.bashrc"
    else
      rc="~/.profile"
    fi
    info "Add to ${rc}: export PATH=\"\$HOME/.local/bin:\$PATH\""
  fi

  # WSL-specific note
  if [[ "$os" == "wsl" ]]; then
    info "Running on WSL: the binary runs natively inside WSL"
    info "3proxy will also run inside WSL (Linux binary)"
  fi

  info "Run 'rip --help' to get started!"
}

# Install 3proxy
install_3proxy() {
  local os=$1
  local arch=$2

  info "Installing 3proxy..."

  local tmp_dir
  tmp_dir=$(mktemp -d)
  cd "$tmp_dir" || exit

  case "$os" in
    linux|wsl)
      # Try official .tar.gz first
      local file="3proxy-${arch}.tar.gz"
      local url="https://github.com/3proxy/3proxy/releases/download/0.9.6/${file}"
      if curl -fsSL -o "$file" "$url" && tar -xzf "$file"; then
        mv 3proxy "${INSTALL_DIR}/3proxy" && chmod +x "${INSTALL_DIR}/3proxy" && \
          success "3proxy installed to ${INSTALL_DIR}/3proxy"
      else
        warn "Could not download 3proxy binary automatically."
        info "Install manually:"
        info "  Debian/Ubuntu: sudo apt install 3proxy"
        info "  RHEL/Fedora:   sudo dnf install 3proxy"
        info "  Alpine:        sudo apk add 3proxy"
        info "  Or build from source: https://github.com/3proxy/3proxy"
      fi
      ;;
    darwin)
      if command -v brew >/dev/null 2>&1; then
        if brew install 3proxy 2>/dev/null; then
          success "3proxy installed via Homebrew"
        else
          warn "Homebrew install failed; try: brew install 3proxy"
        fi
      else
        warn "Homebrew not found."
        info "Install 3proxy on macOS:"
        info "  1. Install Homebrew: /bin/bash -c \"\$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\""
        info "  2. brew install 3proxy"
        info "Or download from: https://github.com/3proxy/3proxy/releases"
      fi
      ;;
    windows)
      warn "3proxy Windows installation not supported by this script."
      info "Download from: https://github.com/3proxy/3proxy/releases"
      ;;
  esac

  rm -rf "$tmp_dir"
  cd - >/dev/null || true
}

# Get latest release version
get_latest_version() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | \
    grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Print colored messages
info()    { echo -e "\033[1;34m[INFO]\033[0m $1"; }
warn()    { echo -e "\033[1;33m[WARN]\033[0m $1"; }
error()   { echo -e "\033[1;31m[ERROR]\033[0m $1" >&2; }
success() { echo -e "\033[1;32m[OK]\033[0m $1"; }

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

  info "Detected: ${os} ($(uname -m) / ${arch})"

  install "$os" "$arch"

  # Optionally install 3proxy
  if [[ "${INSTALL_3PROXY:-false}" == "true" ]]; then
    install_3proxy "$os" "$arch"
  fi
}

main "$@"
