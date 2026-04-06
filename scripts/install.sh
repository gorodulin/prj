#!/bin/sh
#
# Install script for prj (Projector).
# https://github.com/gorodulin/prj
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/gorodulin/prj/main/scripts/install.sh | sh
#   VERSION=0.3.0 curl -sSfL ... | sh
#   INSTALL_DIR=/opt/bin curl -sSfL ... | sh
#
# Works on Linux, macOS, FreeBSD. Compatible with Alpine, slim, and
# other minimal Docker images.

set -eu

REPO="gorodulin/prj"
BINARY_NAME="prj"

# ── Utility functions ────────────────────────────────────────────

info()       { printf '  [*] %s\n' "$1"; }
warn()       { printf '  [!] %s\n' "$1" >&2; }
error_exit() { printf '  [x] %s\n' "$1" >&2; exit 1; }

# ── OS detection ─────────────────────────────────────────────────

detect_os() {
    case "$(uname -s)" in
        Linux*)   echo "linux" ;;
        Darwin*)  echo "darwin" ;;
        FreeBSD*) echo "freebsd" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)        error_exit "Unsupported operating system: $(uname -s)" ;;
    esac
}

# ── Architecture detection ───────────────────────────────────────

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        *)              error_exit "Unsupported architecture: $(uname -m)" ;;
    esac
}

# ── Platform validation ──────────────────────────────────────────

validate_platform() {
    case "${1}/${2}" in
        darwin/amd64|darwin/arm64) ;;
        linux/amd64|linux/arm64) ;;
        freebsd/amd64) ;;
        windows/amd64) ;;
        *) error_exit "Unsupported platform: ${1}/${2}" ;;
    esac
}

# ── Download tool detection ──────────────────────────────────────

detect_downloader() {
    if command -v curl >/dev/null 2>&1; then
        echo "curl"
    elif command -v wget >/dev/null 2>&1; then
        echo "wget"
    else
        error_exit "Neither curl nor wget found. Install one first:
    Alpine:  apk add --no-cache curl
    Debian:  apt-get install -y curl
    RHEL:    yum install -y curl"
    fi
}

# Download a URL to a file.
# $1 = URL, $2 = output path
download() {
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL -o "$2" "$1"
    else
        wget -qO "$2" "$1"
    fi
}

# Fetch a URL and print the response body to stdout.
download_text() {
    if [ "$DOWNLOADER" = "curl" ]; then
        curl -fsSL "$1"
    else
        wget -qO- "$1"
    fi
}

# ── Resolve latest version ───────────────────────────────────────

resolve_version() {
    if [ -n "${VERSION:-}" ]; then
        # Strip leading v if the user passed one.
        VERSION="${VERSION#v}"
        echo "$VERSION"
        return
    fi

    # Use the GitHub API to find the latest release tag.
    TAG=$(download_text "https://api.github.com/repos/${REPO}/releases/latest" \
        | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')

    if [ -z "$TAG" ]; then
        error_exit "Could not determine latest version. Set VERSION explicitly."
    fi

    echo "${TAG#v}"
}

# ── Check for existing package-manager install ───────────────────

check_existing_install() {
    EXISTING="$(command -v "$BINARY_NAME" 2>/dev/null || true)"
    if [ -z "$EXISTING" ]; then
        return
    fi

    # Homebrew
    if command -v brew >/dev/null 2>&1 && brew list "$BINARY_NAME" >/dev/null 2>&1; then
        warn "prj is currently installed via Homebrew."
        warn "Consider: brew uninstall prj"
        warn "Continuing will install a separate copy."
    fi

    # MacPorts
    if command -v port >/dev/null 2>&1 && port installed "$BINARY_NAME" 2>/dev/null | grep -q "$BINARY_NAME"; then
        warn "prj is currently installed via MacPorts."
        warn "Consider: sudo port uninstall prj"
        warn "Continuing will install a separate copy."
    fi
}

# ── Determine install directory ──────────────────────────────────

choose_install_dir() {
    if [ -n "${INSTALL_DIR:-}" ]; then
        echo "$INSTALL_DIR"
        return
    fi

    if [ "$(id -u)" = "0" ]; then
        echo "/usr/local/bin"
        return
    fi

    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    echo "${HOME}/.local/bin"
}

# ── Main ─────────────────────────────────────────────────────────

OS="$(detect_os)"
ARCH="$(detect_arch)"
validate_platform "$OS" "$ARCH"
DOWNLOADER="$(detect_downloader)"

echo "Installing prj for ${OS}/${ARCH}..."

VERSION="$(resolve_version)"
TAG="v${VERSION}"
info "Version: ${VERSION}"

check_existing_install

# Capture the old version for upgrade messaging.
OLD_VERSION=""
if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    OLD_VERSION="$("$BINARY_NAME" --version 2>/dev/null | awk '{print $NF}' || true)"
fi

# Build download URL.
FILENAME="${BINARY_NAME}-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    FILENAME="${FILENAME}.exe"
fi
URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"

# Download to a temp directory.
TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

info "Downloading ${URL}"
download "$URL" "${TMPDIR}/${BINARY_NAME}" || error_exit "Download failed. Check that version ${VERSION} exists and has release assets."
chmod +x "${TMPDIR}/${BINARY_NAME}"

# Install.
INSTALL_DIR="$(choose_install_dir)"
mkdir -p "$INSTALL_DIR"

mv "${TMPDIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
info "Installed to ${INSTALL_DIR}/${BINARY_NAME}"

# Verify.
INSTALLED_VERSION="$("${INSTALL_DIR}/${BINARY_NAME}" --version 2>&1 | awk '{print $NF}' || true)"

if [ -n "$OLD_VERSION" ] && [ "$OLD_VERSION" != "$INSTALLED_VERSION" ]; then
    info "Upgraded prj from ${OLD_VERSION} to ${INSTALLED_VERSION}"
elif [ -n "$OLD_VERSION" ] && [ "$OLD_VERSION" = "$INSTALLED_VERSION" ]; then
    info "prj ${INSTALLED_VERSION} is already up to date"
else
    info "Installed prj ${INSTALLED_VERSION}"
fi

# ── Shell completions ───────────────────────────────────────────

install_completions() {
    PRJ="${INSTALL_DIR}/${BINARY_NAME}"
    COMP=""

    # Bash
    for d in /usr/local/share/bash-completion/completions /usr/share/bash-completion/completions /etc/bash_completion.d; do
        if [ -d "$d" ] && [ -w "$d" ]; then
            "$PRJ" completion bash > "$d/prj" 2>/dev/null && COMP="${COMP} bash"
            break
        fi
    done
    if ! echo "$COMP" | grep -q bash && command -v bash >/dev/null 2>&1; then
        d="${HOME}/.local/share/bash-completion/completions"
        mkdir -p "$d" 2>/dev/null && "$PRJ" completion bash > "$d/prj" 2>/dev/null && COMP="${COMP} bash"
    fi

    # Zsh
    ZSH_HINT=""
    for d in /usr/local/share/zsh/site-functions /usr/share/zsh/site-functions; do
        if [ -d "$d" ] && [ -w "$d" ]; then
            "$PRJ" completion zsh > "$d/_prj" 2>/dev/null && COMP="${COMP} zsh"
            break
        fi
    done
    if ! echo "$COMP" | grep -q zsh && command -v zsh >/dev/null 2>&1; then
        d="${HOME}/.local/share/zsh/site-functions"
        if mkdir -p "$d" 2>/dev/null && "$PRJ" completion zsh > "$d/_prj" 2>/dev/null; then
            COMP="${COMP} zsh"
            ZSH_HINT="$d"
        fi
    fi

    # Fish
    FISH_DONE=false
    for d in /usr/local/share/fish/vendor_completions.d /usr/share/fish/vendor_completions.d; do
        if [ -d "$d" ] && [ -w "$d" ]; then
            "$PRJ" completion fish > "$d/prj.fish" 2>/dev/null && FISH_DONE=true && COMP="${COMP} fish"
            break
        fi
    done
    if ! "$FISH_DONE" && command -v fish >/dev/null 2>&1; then
        d="${HOME}/.config/fish/completions"
        mkdir -p "$d" 2>/dev/null && "$PRJ" completion fish > "$d/prj.fish" 2>/dev/null && COMP="${COMP} fish"
    fi

    if [ -n "$COMP" ]; then
        info "Shell completions installed:${COMP}"
        if [ -n "$ZSH_HINT" ]; then
            info "Zsh: add this to your .zshrc (before compinit):"
            info "  fpath=(${ZSH_HINT} \$fpath)"
        fi
        info "Restart your shell to activate"
    else
        info "To enable completions, see: prj completion --help"
    fi
}

install_completions

# Warn if install dir is not in PATH.
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) warn "${INSTALL_DIR} is not in your PATH. Add it:"
       warn "  export PATH=\"${INSTALL_DIR}:\$PATH\"" ;;
esac
