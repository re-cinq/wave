#!/usr/bin/env bash
# BMAD Review - Initialize code review checklist

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
STORY_ID=""
PR_URL=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --story) STORY_ID="$2"; shift 2 ;;
        --pr) PR_URL="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--json] [--story <id>] [--pr <url>]"
            echo ""
            echo "BMAD Review: Initialize code review checklist"
            echo ""
            echo "Options:"
            echo "  --json         Output in JSON format"
            echo "  --story <id>   Story ID being reviewed"
            echo "  --pr <url>     PR URL to review"
            echo "  --help         Show this help message"
            exit 0
            ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

BMAD_DIR="$REPO_ROOT/.bmad"
REVIEWS_DIR="$BMAD_DIR/reviews"
mkdir -p "$REVIEWS_DIR"

# Generate review ID
REVIEW_ID=$(date +%Y%m%d-%H%M%S)
REVIEW_FILE="$REVIEWS_DIR/review-$REVIEW_ID.md"

# Copy template
TEMPLATE="$REPO_ROOT/.specify/templates/bmad/review-checklist.md"
if [ -f "$TEMPLATE" ]; then
    cp "$TEMPLATE" "$REVIEW_FILE"
else
    # Create minimal review file if template doesn't exist
    cat > "$REVIEW_FILE" << 'EOF'
# Code Review Checklist

**Date**: [DATE]
**Status**: Pending

## Checklist
- [ ] Code compiles
- [ ] Tests pass
- [ ] No security issues
- [ ] Documentation updated
EOF
fi

# Replace placeholders
sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$REVIEW_FILE"

if [ -n "$STORY_ID" ]; then
    sed -i "s/\[FEATURE\/STORY NAME\]/$STORY_ID/g" "$REVIEW_FILE"
fi

if [ -n "$PR_URL" ]; then
    sed -i "s|\[Link to PR\]|$PR_URL|g" "$REVIEW_FILE"
fi

# Get git diff stats if in a git repo
if has_git; then
    DIFF_STATS=$(git diff --stat HEAD~1 2>/dev/null || echo "N/A")
    FILES_CHANGED=$(git diff --name-only HEAD~1 2>/dev/null | wc -l || echo "0")
    LINES_ADDED=$(git diff --numstat HEAD~1 2>/dev/null | awk '{sum+=$1} END {print sum}' || echo "0")
    LINES_REMOVED=$(git diff --numstat HEAD~1 2>/dev/null | awk '{sum+=$2} END {print sum}' || echo "0")
else
    FILES_CHANGED="N/A"
    LINES_ADDED="N/A"
    LINES_REMOVED="N/A"
fi

if $JSON_MODE; then
    printf '{"REVIEW_ID":"%s","REVIEW_FILE":"%s","STORY_ID":"%s","PR_URL":"%s","FILES_CHANGED":"%s","LINES_ADDED":"%s","LINES_REMOVED":"%s"}\n' \
        "$REVIEW_ID" "$REVIEW_FILE" "$STORY_ID" "$PR_URL" "$FILES_CHANGED" "$LINES_ADDED" "$LINES_REMOVED"
else
    echo "REVIEW_ID: $REVIEW_ID"
    echo "REVIEW_FILE: $REVIEW_FILE"
    echo "STORY_ID: $STORY_ID"
    echo "PR_URL: $PR_URL"
    echo ""
    echo "Stats:"
    echo "  Files changed: $FILES_CHANGED"
    echo "  Lines added: $LINES_ADDED"
    echo "  Lines removed: $LINES_REMOVED"
    echo ""
    echo "Review checklist created at: $REVIEW_FILE"
fi
