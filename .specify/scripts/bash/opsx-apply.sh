#!/usr/bin/env bash
# OpenSpec Apply - Execute implementation tasks

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
CHANGE_SLUG=""
TASK_ID=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --change) CHANGE_SLUG="$2"; shift 2 ;;
        --task) TASK_ID="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--json] [--change <slug>] [--task <id>]"
            echo ""
            echo "OpenSpec Apply: Execute implementation tasks"
            echo ""
            echo "Options:"
            echo "  --json           Output in JSON format"
            echo "  --change <slug>  Change slug (auto-detect if single change)"
            echo "  --task <id>      Specific task ID to execute"
            echo "  --help           Show this help message"
            exit 0
            ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

OPENSPEC_DIR="$REPO_ROOT/openspec"

# Find change directory
if [ -z "$CHANGE_SLUG" ]; then
    CHANGE_COUNT=$(find "$OPENSPEC_DIR/changes" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [ "$CHANGE_COUNT" -eq 2 ]; then
        CHANGE_SLUG=$(ls "$OPENSPEC_DIR/changes" | head -1)
    elif [ "$CHANGE_COUNT" -gt 2 ]; then
        echo "Error: Multiple changes found. Please specify --change <slug>" >&2
        echo "Available changes:" >&2
        ls "$OPENSPEC_DIR/changes" >&2
        exit 1
    else
        echo "Error: No changes found. Run /opsx.new first." >&2
        exit 1
    fi
fi

CHANGE_DIR="$OPENSPEC_DIR/changes/$CHANGE_SLUG"

if [ ! -d "$CHANGE_DIR" ]; then
    echo "Error: Change not found: $CHANGE_SLUG" >&2
    exit 1
fi

# Check required files exist
DESIGN_FILE="$CHANGE_DIR/design.md"
TASKS_FILE="$CHANGE_DIR/tasks.md"

if [ ! -f "$DESIGN_FILE" ]; then
    echo "Error: Design doc not found. Run /opsx.ff first." >&2
    exit 1
fi

if [ ! -f "$TASKS_FILE" ]; then
    echo "Error: Tasks file not found. Run /opsx.ff first." >&2
    exit 1
fi

if $JSON_MODE; then
    printf '{"CHANGE_SLUG":"%s","CHANGE_DIR":"%s","DESIGN_FILE":"%s","TASKS_FILE":"%s","TASK_ID":"%s"}\n' \
        "$CHANGE_SLUG" "$CHANGE_DIR" "$DESIGN_FILE" "$TASKS_FILE" "$TASK_ID"
else
    echo "CHANGE_SLUG: $CHANGE_SLUG"
    echo "CHANGE_DIR: $CHANGE_DIR"
    echo "DESIGN_FILE: $DESIGN_FILE"
    echo "TASKS_FILE: $TASKS_FILE"
    if [ -n "$TASK_ID" ]; then
        echo "TASK_ID: $TASK_ID"
    fi
    echo ""
    echo "Ready to apply changes. The AI agent will:"
    echo "  1. Read the tasks from $TASKS_FILE"
    echo "  2. Execute tasks in dependency order"
    echo "  3. Update task status as work progresses"
fi
