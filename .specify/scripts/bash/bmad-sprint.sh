#!/usr/bin/env bash
# BMAD Sprint - Initialize sprint tracking

set -e

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

JSON_MODE=false
SPRINT_NUM=""
PRODUCT_SLUG=""
ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --json) JSON_MODE=true; shift ;;
        --sprint) SPRINT_NUM="$2"; shift 2 ;;
        --product) PRODUCT_SLUG="$2"; shift 2 ;;
        --help|-h)
            echo "Usage: $0 [--json] [--sprint <num>] [--product <slug>]"
            echo ""
            echo "BMAD Sprint: Initialize sprint tracking"
            echo ""
            echo "Options:"
            echo "  --json           Output in JSON format"
            echo "  --sprint <num>   Sprint number (auto-increments if not specified)"
            echo "  --product <slug> Product slug (required if multiple products exist)"
            echo "  --help           Show this help message"
            exit 0
            ;;
        *) ARGS+=("$1"); shift ;;
    esac
done

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

BMAD_DIR="$REPO_ROOT/.bmad"

# Find product directory
if [ -z "$PRODUCT_SLUG" ]; then
    # Check if there's only one product
    PRODUCT_COUNT=$(find "$BMAD_DIR/products" -maxdepth 1 -type d 2>/dev/null | wc -l)
    if [ "$PRODUCT_COUNT" -eq 2 ]; then
        PRODUCT_SLUG=$(ls "$BMAD_DIR/products" | head -1)
    elif [ "$PRODUCT_COUNT" -gt 2 ]; then
        echo "Error: Multiple products found. Please specify --product <slug>" >&2
        echo "Available products:" >&2
        ls "$BMAD_DIR/products" >&2
        exit 1
    else
        echo "Error: No products found. Run /bmad.product-brief first." >&2
        exit 1
    fi
fi

PRODUCT_DIR="$BMAD_DIR/products/$PRODUCT_SLUG"
SPRINTS_DIR="$PRODUCT_DIR/sprints"

if [ ! -d "$PRODUCT_DIR" ]; then
    echo "Error: Product not found: $PRODUCT_SLUG" >&2
    exit 1
fi

mkdir -p "$SPRINTS_DIR"

# Determine sprint number
if [ -z "$SPRINT_NUM" ]; then
    HIGHEST=0
    for dir in "$SPRINTS_DIR"/sprint-*; do
        [ -d "$dir" ] || continue
        num=$(basename "$dir" | sed 's/sprint-//')
        if [ "$num" -gt "$HIGHEST" ] 2>/dev/null; then
            HIGHEST=$num
        fi
    done
    SPRINT_NUM=$((HIGHEST + 1))
fi

SPRINT_DIR="$SPRINTS_DIR/sprint-$SPRINT_NUM"
mkdir -p "$SPRINT_DIR"

# Create sprint tracking file
SPRINT_FILE="$SPRINT_DIR/sprint.md"
cat > "$SPRINT_FILE" << EOF
# Sprint $SPRINT_NUM

**Product**: $PRODUCT_SLUG
**Start Date**: $(date +%Y-%m-%d)
**End Date**: [TBD]
**Status**: Planning

## Sprint Goal

[Define the sprint goal - what will be delivered?]

## Capacity

| Team Member | Available Days | Planned Points |
|-------------|----------------|----------------|
| [Name] | [X] | [Y] |
| **Total** | **[X]** | **[Y]** |

## Committed Stories

| ID | Story | Points | Owner | Status |
|----|-------|--------|-------|--------|
| | | | | Pending |

## Sprint Backlog

### In Progress
- [ ] [Story ID]: [Story title]

### To Do
- [ ] [Story ID]: [Story title]

### Done
- [x] [Story ID]: [Story title]

## Daily Standups

### Day 1 - $(date +%Y-%m-%d)
**Progress**: [What was done]
**Plan**: [What's next]
**Blockers**: [Any blockers]

## Retrospective

### What went well
- [Item]

### What could be improved
- [Item]

### Action items
- [ ] [Action]

## Metrics

- **Planned Points**: [X]
- **Completed Points**: [Y]
- **Velocity**: [Z]
- **Burndown**: [Link or inline]
EOF

if $JSON_MODE; then
    printf '{"SPRINT_NUM":%d,"SPRINT_DIR":"%s","SPRINT_FILE":"%s","PRODUCT_SLUG":"%s"}\n' \
        "$SPRINT_NUM" "$SPRINT_DIR" "$SPRINT_FILE" "$PRODUCT_SLUG"
else
    echo "SPRINT_NUM: $SPRINT_NUM"
    echo "SPRINT_DIR: $SPRINT_DIR"
    echo "SPRINT_FILE: $SPRINT_FILE"
    echo "PRODUCT_SLUG: $PRODUCT_SLUG"
    echo ""
    echo "Sprint $SPRINT_NUM initialized. Next steps:"
    echo "  1. Define sprint goal in $SPRINT_FILE"
    echo "  2. Add committed stories from epics"
    echo "  3. Run /bmad.story to create detailed stories"
fi
