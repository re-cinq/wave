#!/usr/bin/env bash

set -e

JSON_MODE=false
FEATURE_NUMBER=""
SHORT_NAME=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --number) FEATURE_NUMBER="$2"; shift 2 ;;
        --short-name) SHORT_NAME="$2"; shift 2 ;;
        --help|-h) echo "Usage: $0 [--json] [--number NUM] [--short-name NAME] <feature_description>"; exit 0 ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

FEATURE_DESCRIPTION="${ARGS[*]}"
if [ -z "$FEATURE_DESCRIPTION" ]; then
    echo "Usage: $0 [--json] [--number NUM] [--short-name NAME] <feature_description>" >&2
    echo "Examples:" >&2
    echo "  $0 --json --number 5 --short-name 'user-auth' 'Add user authentication'" >&2
    echo "  $0 'Create analytics dashboard'" >&2
    exit 1
fi

# Function to find the repository root by searching for existing project markers
find_repo_root() {
    local dir="$1"
    while [ "$dir" != "/" ]; do
        if [ -d "$dir/.git" ] || [ -d "$dir/.specify" ]; then
            echo "$dir"
            return 0
        fi
        dir="$(dirname "$dir")"
    done
    return 1
}

# Resolve repository root. Prefer git information when available, but fall back
# to searching for repository markers so the workflow still functions in repositories that
# were initialised with --no-git.
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if git rev-parse --show-toplevel >/dev/null 2>&1; then
    REPO_ROOT=$(git rev-parse --show-toplevel)
    HAS_GIT=true
else
    REPO_ROOT="$(find_repo_root "$SCRIPT_DIR")"
    if [ -z "$REPO_ROOT" ]; then
        echo "Error: Could not determine repository root. Please run this script from within the repository." >&2
        exit 1
    fi
    HAS_GIT=false
fi

cd "$REPO_ROOT"

SPECS_DIR="$REPO_ROOT/specs"
mkdir -p "$SPECS_DIR"

# Use provided feature number or auto-calculate
if [ -n "$FEATURE_NUMBER" ]; then
    FEATURE_NUM=$(printf "%03d" "$FEATURE_NUMBER")
    NEXT="$FEATURE_NUMBER"
else
    # Auto-calculate next available number
    HIGHEST=0
    if [ -d "$SPECS_DIR" ]; then
        for dir in "$SPECS_DIR"/*; do
            [ -d "$dir" ] || continue
            dirname=$(basename "$dir")
            number=$(echo "$dirname" | grep -o '^[0-9]\+' || echo "0")
            number=$((10#$number))
            if [ "$number" -gt "$HIGHEST" ]; then HIGHEST=$number; fi
        done
    fi

    NEXT=$((HIGHEST + 1))
    FEATURE_NUM=$(printf "%03d" "$NEXT")
fi

# Use provided short name or auto-generate from description
if [ -n "$SHORT_NAME" ]; then
    BRANCH_NAME="${FEATURE_NUM}-${SHORT_NAME}"
else
    # Auto-generate short name from feature description
    AUTO_NAME=$(echo "$FEATURE_DESCRIPTION" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//' | sed 's/-$//')
    WORDS=$(echo "$AUTO_NAME" | tr '-' '\n' | grep -v '^$' | head -3 | tr '\n' '-' | sed 's/-$//')
    BRANCH_NAME="${FEATURE_NUM}-${WORDS}"
fi

if [ "$HAS_GIT" = true ]; then
    # Create branch without switching — worktrees handle checkout isolation
    git branch "$BRANCH_NAME" 2>/dev/null || true
else
    >&2 echo "[specify] Warning: Git repository not detected; skipped branch creation for $BRANCH_NAME"
fi

FEATURE_DIR="$SPECS_DIR/$BRANCH_NAME"
mkdir -p "$FEATURE_DIR"

TEMPLATE="$REPO_ROOT/.specify/templates/spec-template.md"
SPEC_FILE="$FEATURE_DIR/spec.md"
if [ -f "$TEMPLATE" ]; then cp "$TEMPLATE" "$SPEC_FILE"; else touch "$SPEC_FILE"; fi

# Set the SPECIFY_FEATURE environment variable for the current session
export SPECIFY_FEATURE="$BRANCH_NAME"

if $JSON_MODE; then
    printf '{"BRANCH_NAME":"%s","SPEC_FILE":"%s","FEATURE_NUM":"%s"}\n' "$BRANCH_NAME" "$SPEC_FILE" "$FEATURE_NUM"
else
    echo "BRANCH_NAME: $BRANCH_NAME"
    echo "SPEC_FILE: $SPEC_FILE"
    echo "FEATURE_NUM: $FEATURE_NUM"
    echo "SPECIFY_FEATURE environment variable set to: $BRANCH_NAME"
fi
