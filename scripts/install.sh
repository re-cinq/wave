#!/bin/sh
set -eu

# Cross-platform install script for Wave
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
#   curl -fsSL ... | sh -s -- 0.1.0
#
# Environment variables:
#   WAVE_INSTALL_DIR  Override the default install directory

REPO="re-cinq/wave"
BINARY="wave"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

log() {
    printf '==> %s\n' "$1"
}

err() {
    printf '==> ERROR: %s\n' "$1" >&2
    exit 1
}

need_cmd() {
    if ! command -v "$1" >/dev/null 2>&1; then
        err "required command not found: $1"
    fi
}

# ---------------------------------------------------------------------------
# Detection
# ---------------------------------------------------------------------------

detect_os() {
    _detect_os_uname="$(uname -s)"
    case "$_detect_os_uname" in
        Linux)  OS="linux" ;;
        Darwin) OS="darwin" ;;
        *)      err "unsupported operating system: $_detect_os_uname" ;;
    esac
}

detect_arch() {
    _detect_arch_uname="$(uname -m)"
    case "$_detect_arch_uname" in
        x86_64)          ARCH="amd64" ;;
        aarch64|arm64)   ARCH="arm64" ;;
        *)               err "unsupported architecture: $_detect_arch_uname" ;;
    esac
}

detect_install_dir() {
    if [ -n "${WAVE_INSTALL_DIR:-}" ]; then
        INSTALL_DIR="$WAVE_INSTALL_DIR"
    elif [ "$(id -u)" = "0" ]; then
        INSTALL_DIR="/usr/local/bin"
    else
        INSTALL_DIR="${HOME}/.local/bin"
    fi
}

# ---------------------------------------------------------------------------
# Version resolution
# ---------------------------------------------------------------------------

resolve_version() {
    _resolve_version_input="${1:-latest}"

    if [ "$_resolve_version_input" != "latest" ]; then
        # Strip leading "v" if the caller passed one
        VERSION="$(printf '%s' "$_resolve_version_input" | sed 's/^v//')"
        return
    fi

    log "resolving latest version from GitHub..."
    _resolve_version_response="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")" \
        || err "failed to query GitHub API for latest release"

    VERSION="$(printf '%s' "$_resolve_version_response" \
        | grep '"tag_name"' \
        | sed -E 's/.*"v([^"]+)".*/\1/')"

    if [ -z "$VERSION" ]; then
        err "could not determine latest version from GitHub API response"
    fi
}

# ---------------------------------------------------------------------------
# Download and verify
# ---------------------------------------------------------------------------

download_and_verify() {
    case "$OS" in
        linux)  _dav_ext="tar.gz" ;;
        darwin) _dav_ext="zip" ;;
        *)      err "unsupported OS for archive format: $OS" ;;
    esac

    _dav_archive="${BINARY}_${VERSION}_${OS}_${ARCH}.${_dav_ext}"
    _dav_checksums="checksums.txt"
    _dav_base_url="https://github.com/${REPO}/releases/download/v${VERSION}"

    _dav_archive_url="${_dav_base_url}/${_dav_archive}"
    _dav_checksums_url="${_dav_base_url}/${_dav_checksums}"

    log "downloading ${_dav_archive}..."
    curl -fSL -o "${TMPDIR_INSTALL}/${_dav_archive}" "$_dav_archive_url" \
        || err "failed to download archive: ${_dav_archive_url}"

    log "downloading checksums..."
    curl -fSL -o "${TMPDIR_INSTALL}/${_dav_checksums}" "$_dav_checksums_url" \
        || err "failed to download checksums: ${_dav_checksums_url}"

    log "verifying checksum..."
    _dav_expected="$(grep "${_dav_archive}" "${TMPDIR_INSTALL}/${_dav_checksums}" | awk '{print $1}')"
    if [ -z "$_dav_expected" ]; then
        err "archive ${_dav_archive} not found in checksums file"
    fi

    case "$OS" in
        linux)
            _dav_actual="$(cd "${TMPDIR_INSTALL}" && sha256sum "${_dav_archive}" | awk '{print $1}')"
            ;;
        darwin)
            _dav_actual="$(cd "${TMPDIR_INSTALL}" && shasum -a 256 "${_dav_archive}" | awk '{print $1}')"
            ;;
    esac

    if [ "$_dav_expected" != "$_dav_actual" ]; then
        err "checksum mismatch: expected ${_dav_expected}, got ${_dav_actual}"
    fi
    log "checksum verified"

    # Extract
    log "extracting binary..."
    case "$_dav_ext" in
        tar.gz)
            tar -xzf "${TMPDIR_INSTALL}/${_dav_archive}" -C "${TMPDIR_INSTALL}" "$BINARY" \
                || err "failed to extract archive"
            ;;
        zip)
            need_cmd unzip
            unzip -o -q "${TMPDIR_INSTALL}/${_dav_archive}" "$BINARY" -d "${TMPDIR_INSTALL}" \
                || err "failed to extract archive"
            ;;
    esac

    if [ ! -f "${TMPDIR_INSTALL}/${BINARY}" ]; then
        err "binary not found after extraction"
    fi
}

# ---------------------------------------------------------------------------
# Install
# ---------------------------------------------------------------------------

install_binary() {
    log "installing to ${INSTALL_DIR}/${BINARY}..."
    mkdir -p "$INSTALL_DIR" || err "failed to create install directory: ${INSTALL_DIR}"
    cp "${TMPDIR_INSTALL}/${BINARY}" "${INSTALL_DIR}/${BINARY}" \
        || err "failed to copy binary to ${INSTALL_DIR}/${BINARY}"
    chmod 755 "${INSTALL_DIR}/${BINARY}" \
        || err "failed to set permissions on ${INSTALL_DIR}/${BINARY}"
}

# ---------------------------------------------------------------------------
# Cleanup
# ---------------------------------------------------------------------------

cleanup() {
    if [ -n "${TMPDIR_INSTALL:-}" ] && [ -d "${TMPDIR_INSTALL}" ]; then
        rm -rf "${TMPDIR_INSTALL}"
    fi
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

main() {
    need_cmd curl
    need_cmd tar
    need_cmd uname

    detect_os
    detect_arch
    detect_install_dir

    resolve_version "${1:-latest}"

    log "installing ${BINARY} v${VERSION} (${OS}/${ARCH})"

    TMPDIR_INSTALL="$(mktemp -d)" || err "failed to create temporary directory"
    trap cleanup EXIT

    download_and_verify
    install_binary

    log "successfully installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"

    # Warn if install directory is not on PATH
    case ":${PATH}:" in
        *":${INSTALL_DIR}:"*) ;;
        *)
            printf '\n'
            log "NOTE: ${INSTALL_DIR} is not in your PATH"
            log "Add it by running:"
            log "  export PATH=\"${INSTALL_DIR}:\$PATH\""
            printf '\n'
            ;;
    esac
}

main "$@"
