#!/bin/sh
# Wave CLI Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/recinq/wave/main/install.sh | sh
#
# Environment variables:
#   WAVE_INSTALL_DIR  - Installation directory (default: /usr/local/bin or ~/.local/bin)
#   WAVE_VERSION      - Specific version to install (default: latest)
#   WAVE_NO_MODIFY_PATH - Set to 1 to skip PATH modification

set -e

# Colors for output (disabled if not a terminal)
if [ -t 1 ]; then
    RED='\033[0;31m'
    GREEN='\033[0;32m'
    YELLOW='\033[0;33m'
    BLUE='\033[0;34m'
    BOLD='\033[1m'
    NC='\033[0m' # No Color
else
    RED=''
    GREEN=''
    YELLOW=''
    BLUE=''
    BOLD=''
    NC=''
fi

GITHUB_REPO="recinq/wave"
BINARY_NAME="wave"

# Print colored output
info() {
    printf "${BLUE}==>${NC} ${BOLD}%s${NC}\n" "$1"
}

success() {
    printf "${GREEN}==>${NC} ${BOLD}%s${NC}\n" "$1"
}

warn() {
    printf "${YELLOW}Warning:${NC} %s\n" "$1"
}

error() {
    printf "${RED}Error:${NC} %s\n" "$1" >&2
    exit 1
}

# Detect OS
detect_os() {
    OS="$(uname -s)"
    case "$OS" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          error "Unsupported operating system: $OS" ;;
    esac
}

# Detect architecture
detect_arch() {
    ARCH="$(uname -m)"
    case "$ARCH" in
        x86_64|amd64)   echo "amd64" ;;
        aarch64|arm64)  echo "arm64" ;;
        armv7l)         echo "arm" ;;
        i386|i686)      echo "386" ;;
        *)              error "Unsupported architecture: $ARCH" ;;
    esac
}

# Get the latest release version from GitHub
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${GITHUB_REPO}/releases/latest" | \
            grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Download file
download() {
    URL="$1"
    OUTPUT="$2"

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$URL" -o "$OUTPUT"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$URL" -O "$OUTPUT"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi
}

# Determine install directory
get_install_dir() {
    if [ -n "$WAVE_INSTALL_DIR" ]; then
        echo "$WAVE_INSTALL_DIR"
        return
    fi

    # Try /usr/local/bin first (requires sudo)
    if [ -w "/usr/local/bin" ]; then
        echo "/usr/local/bin"
        return
    fi

    # Fall back to ~/.local/bin
    LOCAL_BIN="$HOME/.local/bin"
    mkdir -p "$LOCAL_BIN"
    echo "$LOCAL_BIN"
}

# Check if directory is in PATH
is_in_path() {
    case ":$PATH:" in
        *":$1:"*) return 0 ;;
        *) return 1 ;;
    esac
}

# Add directory to PATH in shell config
add_to_path() {
    DIR="$1"

    if [ "$WAVE_NO_MODIFY_PATH" = "1" ]; then
        return
    fi

    if is_in_path "$DIR"; then
        return
    fi

    SHELL_NAME="$(basename "$SHELL")"
    EXPORT_LINE="export PATH=\"$DIR:\$PATH\""

    case "$SHELL_NAME" in
        bash)
            if [ -f "$HOME/.bashrc" ]; then
                echo "" >> "$HOME/.bashrc"
                echo "# Added by Wave installer" >> "$HOME/.bashrc"
                echo "$EXPORT_LINE" >> "$HOME/.bashrc"
                warn "Added $DIR to PATH in ~/.bashrc"
                warn "Run 'source ~/.bashrc' or start a new terminal to use wave"
            fi
            ;;
        zsh)
            if [ -f "$HOME/.zshrc" ]; then
                echo "" >> "$HOME/.zshrc"
                echo "# Added by Wave installer" >> "$HOME/.zshrc"
                echo "$EXPORT_LINE" >> "$HOME/.zshrc"
                warn "Added $DIR to PATH in ~/.zshrc"
                warn "Run 'source ~/.zshrc' or start a new terminal to use wave"
            fi
            ;;
        fish)
            FISH_CONFIG="$HOME/.config/fish/config.fish"
            if [ -f "$FISH_CONFIG" ]; then
                echo "" >> "$FISH_CONFIG"
                echo "# Added by Wave installer" >> "$FISH_CONFIG"
                echo "set -gx PATH $DIR \$PATH" >> "$FISH_CONFIG"
                warn "Added $DIR to PATH in $FISH_CONFIG"
                warn "Start a new terminal to use wave"
            fi
            ;;
        *)
            warn "$DIR is not in your PATH"
            warn "Add the following to your shell configuration:"
            warn "  $EXPORT_LINE"
            ;;
    esac
}

# Main installation
main() {
    echo ""
    printf "${BOLD}Wave CLI Installer${NC}\n"
    echo "─────────────────────────────────────"
    echo ""

    # Detect platform
    OS="$(detect_os)"
    ARCH="$(detect_arch)"
    info "Detected platform: ${OS}/${ARCH}"

    # Get version
    if [ -n "$WAVE_VERSION" ]; then
        VERSION="$WAVE_VERSION"
    else
        info "Fetching latest version..."
        VERSION="$(get_latest_version)"
        if [ -z "$VERSION" ]; then
            error "Failed to fetch latest version. Set WAVE_VERSION manually or check your internet connection."
        fi
    fi
    info "Version: ${VERSION}"

    # Construct download URL
    # Expected format: wave-{os}-{arch} or wave-{os}-{arch}.exe for Windows
    if [ "$OS" = "windows" ]; then
        BINARY_SUFFIX=".exe"
    else
        BINARY_SUFFIX=""
    fi

    ASSET_NAME="${BINARY_NAME}-${OS}-${ARCH}${BINARY_SUFFIX}"
    DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${VERSION}/${ASSET_NAME}"

    # Create temp directory
    TMP_DIR="$(mktemp -d)"
    trap "rm -rf '$TMP_DIR'" EXIT

    TMP_FILE="${TMP_DIR}/${BINARY_NAME}${BINARY_SUFFIX}"

    # Download binary
    info "Downloading ${ASSET_NAME}..."
    if ! download "$DOWNLOAD_URL" "$TMP_FILE"; then
        error "Failed to download from ${DOWNLOAD_URL}"
    fi

    # Make executable
    chmod +x "$TMP_FILE"

    # Verify binary works
    info "Verifying binary..."
    if ! "$TMP_FILE" --help >/dev/null 2>&1; then
        error "Downloaded binary appears to be invalid"
    fi

    # Get install directory
    INSTALL_DIR="$(get_install_dir)"
    INSTALL_PATH="${INSTALL_DIR}/${BINARY_NAME}${BINARY_SUFFIX}"

    info "Installing to ${INSTALL_PATH}..."

    # Check if we need sudo
    if [ ! -w "$INSTALL_DIR" ]; then
        if command -v sudo >/dev/null 2>&1; then
            sudo mv "$TMP_FILE" "$INSTALL_PATH"
            sudo chmod +x "$INSTALL_PATH"
        else
            error "Cannot write to ${INSTALL_DIR} and sudo is not available"
        fi
    else
        mv "$TMP_FILE" "$INSTALL_PATH"
        chmod +x "$INSTALL_PATH"
    fi

    # Add to PATH if needed
    add_to_path "$INSTALL_DIR"

    echo ""
    success "Wave ${VERSION} installed successfully!"
    echo ""

    # Verify installation
    if command -v wave >/dev/null 2>&1; then
        echo "  Location: $(command -v wave)"
        echo ""
        echo "  Get started:"
        echo "    wave init        # Initialize a new project"
        echo "    wave --help      # Show available commands"
        echo ""
    else
        echo "  Location: ${INSTALL_PATH}"
        echo ""
        if ! is_in_path "$INSTALL_DIR"; then
            warn "wave is not in your PATH yet"
            echo "  Run: export PATH=\"${INSTALL_DIR}:\$PATH\""
            echo ""
        fi
    fi

    # Check for Claude CLI
    if ! command -v claude >/dev/null 2>&1; then
        warn "Claude Code CLI (claude) not found in PATH"
        echo "  Wave requires Claude Code to execute pipelines."
        echo "  Install from: https://claude.ai/code"
        echo ""
    fi
}

main "$@"
