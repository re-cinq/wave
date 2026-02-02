#!/usr/bin/env bash
# OpenSpec Fast-Forward - Generate all planning docs

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
CHANGE_SLUG=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --change) CHANGE_SLUG="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--json] [--change <slug>]"
            echo ""
            echo "OpenSpec Fast-Forward: Generate all planning docs"
            echo ""
            echo "Options:"
            echo "  --json           Output in JSON format"
            echo "  --change <slug>  Change slug (auto-detect if single change)"
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
    # Check if there's only one change
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

# Check proposal exists
PROPOSAL_FILE="$CHANGE_DIR/proposal.md"
if [ ! -f "$PROPOSAL_FILE" ]; then
    echo "Error: Proposal not found: $PROPOSAL_FILE" >&2
    exit 1
fi

# Copy templates
TEMPLATES_DIR="$REPO_ROOT/.specify/templates/openspec"

DESIGN_FILE="$CHANGE_DIR/design.md"
if [ ! -f "$DESIGN_FILE" ] && [ -f "$TEMPLATES_DIR/design.md" ]; then
    cp "$TEMPLATES_DIR/design.md" "$DESIGN_FILE"
    # Extract change name from proposal
    CHANGE_NAME=$(grep -m1 "^# Change Proposal:" "$PROPOSAL_FILE" | sed 's/^# Change Proposal: //' || echo "$CHANGE_SLUG")
    sed -i "s|\[CHANGE NAME\]|$CHANGE_NAME|g" "$DESIGN_FILE"
    sed -i "s|\[Link to proposal.md\]|./proposal.md|g" "$DESIGN_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$DESIGN_FILE"
fi

TASKS_FILE="$CHANGE_DIR/tasks.md"
if [ ! -f "$TASKS_FILE" ] && [ -f "$TEMPLATES_DIR/tasks.md" ]; then
    cp "$TEMPLATES_DIR/tasks.md" "$TASKS_FILE"
    CHANGE_NAME=$(grep -m1 "^# Change Proposal:" "$PROPOSAL_FILE" | sed 's/^# Change Proposal: //' || echo "$CHANGE_SLUG")
    sed -i "s|\[CHANGE NAME\]|$CHANGE_NAME|g" "$TASKS_FILE"
    sed -i "s|\[Link to proposal.md\]|./proposal.md|g" "$TASKS_FILE"
    sed -i "s|\[Link to design.md\]|./design.md|g" "$TASKS_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$TASKS_FILE"
fi

if $JSON_MODE; then
    printf '{"CHANGE_SLUG":"%s","CHANGE_DIR":"%s","PROPOSAL_FILE":"%s","DESIGN_FILE":"%s","TASKS_FILE":"%s"}\n' \
        "$CHANGE_SLUG" "$CHANGE_DIR" "$PROPOSAL_FILE" "$DESIGN_FILE" "$TASKS_FILE"
else
    echo "CHANGE_SLUG: $CHANGE_SLUG"
    echo "CHANGE_DIR: $CHANGE_DIR"
    echo "PROPOSAL_FILE: $PROPOSAL_FILE"
    echo "DESIGN_FILE: $DESIGN_FILE"
    echo "TASKS_FILE: $TASKS_FILE"
    echo ""
    echo "Planning docs generated. Next steps:"
    echo "  1. Complete the design: $DESIGN_FILE"
    echo "  2. Define tasks: $TASKS_FILE"
    echo "  3. Run /opsx.apply to implement"
fi
