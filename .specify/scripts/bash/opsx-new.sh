#!/usr/bin/env bash
# OpenSpec New - Create a new change proposal

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
ARGS=()
for arg in "$@"; do
    case "$arg" in
        --json) JSON_MODE=true ;;
        --help|-h)
            echo "Usage: $0 [--json] <change_name>"
            echo ""
            echo "OpenSpec New: Create a new change proposal"
            echo ""
            echo "Options:"
            echo "  --json    Output in JSON format"
            echo "  --help    Show this help message"
            exit 0
            ;;
        *) ARGS+=("$arg") ;;
    esac
done

CHANGE_NAME="${ARGS[*]}"
if [ -z "$CHANGE_NAME" ]; then
    echo "Usage: $0 [--json] <change_name>" >&2
    exit 1
fi

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

# Create slug from change name
CHANGE_SLUG=$(echo "$CHANGE_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//' | sed 's/-$//')

# Create OpenSpec directory structure
OPENSPEC_DIR="$REPO_ROOT/openspec"
CHANGE_DIR="$OPENSPEC_DIR/changes/$CHANGE_SLUG"

mkdir -p "$CHANGE_DIR/specs"
mkdir -p "$OPENSPEC_DIR/archive"

# Copy proposal template
TEMPLATE="$REPO_ROOT/.specify/templates/openspec/proposal.md"
PROPOSAL_FILE="$CHANGE_DIR/proposal.md"

if [ -f "$TEMPLATE" ]; then
    cp "$TEMPLATE" "$PROPOSAL_FILE"
    sed -i "s/\[CHANGE NAME\]/$CHANGE_NAME/g" "$PROPOSAL_FILE"
    sed -i "s/\[OPSX-XXX\]/OPSX-$(date +%Y%m%d)/g" "$PROPOSAL_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$PROPOSAL_FILE"
else
    # Create minimal proposal if template doesn't exist
    cat > "$PROPOSAL_FILE" << EOF
# Change Proposal: $CHANGE_NAME

**ID**: OPSX-$(date +%Y%m%d)
**Created**: $(date +%Y-%m-%d)
**Status**: Draft

## Summary

[Description of the change]

## Requirements

- [ ] Requirement 1

## Success Criteria

- [ ] Criterion 1
EOF
fi

if $JSON_MODE; then
    printf '{"CHANGE_NAME":"%s","CHANGE_SLUG":"%s","CHANGE_DIR":"%s","PROPOSAL_FILE":"%s"}\n' \
        "$CHANGE_NAME" "$CHANGE_SLUG" "$CHANGE_DIR" "$PROPOSAL_FILE"
else
    echo "CHANGE_NAME: $CHANGE_NAME"
    echo "CHANGE_SLUG: $CHANGE_SLUG"
    echo "CHANGE_DIR: $CHANGE_DIR"
    echo "PROPOSAL_FILE: $PROPOSAL_FILE"
    echo ""
    echo "Next steps:"
    echo "  1. Edit the proposal: $PROPOSAL_FILE"
    echo "  2. Run /opsx.ff to generate planning docs"
fi
