#!/usr/bin/env bash
# OpenSpec Archive - Archive completed changes

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
CHANGE_SLUG=""
FORCE=false
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --change) CHANGE_SLUG="$2"; shift 2 ;;
        --force|-f) FORCE=true; shift ;;
        --help|-h)
            echo "Usage: $0 [--json] [--change <slug>] [--force]"
            echo ""
            echo "OpenSpec Archive: Archive completed changes"
            echo ""
            echo "Options:"
            echo "  --json           Output in JSON format"
            echo "  --change <slug>  Change slug (required)"
            echo "  --force          Skip confirmation"
            echo "  --help           Show this help message"
            exit 0
            ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

if [ -z "$CHANGE_SLUG" ]; then
    echo "Error: --change <slug> is required" >&2
    echo "Available changes:" >&2
    ls "$REPO_ROOT/openspec/changes" 2>/dev/null || echo "  (none)" >&2
    exit 1
fi

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

OPENSPEC_DIR="$REPO_ROOT/openspec"
CHANGE_DIR="$OPENSPEC_DIR/changes/$CHANGE_SLUG"
ARCHIVE_DIR="$OPENSPEC_DIR/archive"

if [ ! -d "$CHANGE_DIR" ]; then
    echo "Error: Change not found: $CHANGE_SLUG" >&2
    exit 1
fi

mkdir -p "$ARCHIVE_DIR"

# Create archive name with date
ARCHIVE_NAME="$(date +%Y%m%d)-$CHANGE_SLUG"
ARCHIVE_PATH="$ARCHIVE_DIR/$ARCHIVE_NAME"

# Check if archive already exists
if [ -d "$ARCHIVE_PATH" ]; then
    echo "Error: Archive already exists: $ARCHIVE_PATH" >&2
    exit 1
fi

# Confirm unless --force
if [ "$FORCE" != "true" ] && [ "$JSON_MODE" != "true" ]; then
    echo "About to archive: $CHANGE_DIR"
    echo "To: $ARCHIVE_PATH"
    read -p "Continue? [y/N] " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        echo "Cancelled."
        exit 0
    fi
fi

# Move to archive
mv "$CHANGE_DIR" "$ARCHIVE_PATH"

if $JSON_MODE; then
    printf '{"CHANGE_SLUG":"%s","ARCHIVE_PATH":"%s","ARCHIVED":true}\n' \
        "$CHANGE_SLUG" "$ARCHIVE_PATH"
else
    echo "CHANGE_SLUG: $CHANGE_SLUG"
    echo "ARCHIVE_PATH: $ARCHIVE_PATH"
    echo ""
    echo "Change archived successfully."
fi
