#!/usr/bin/env bash
# BMAD Setup - Initialize product brief and full planning structure

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
            echo "Usage: $0 [--json] <product_name>"
            echo ""
            echo "BMAD Setup: Initialize product brief and full planning structure"
            echo ""
            echo "Options:"
            echo "  --json    Output in JSON format"
            echo "  --help    Show this help message"
            exit 0
            ;;
        *) ARGS+=("$arg") ;;
    esac
done

PRODUCT_NAME="${ARGS[*]}"
if [ -z "$PRODUCT_NAME" ]; then
    echo "Usage: $0 [--json] <product_name>" >&2
    exit 1
fi

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

# Create slug from product name
PRODUCT_SLUG=$(echo "$PRODUCT_NAME" | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g' | sed 's/-\+/-/g' | sed 's/^-//' | sed 's/-$//')

# Create BMAD directory structure
BMAD_DIR="$REPO_ROOT/.bmad"
PRODUCT_DIR="$BMAD_DIR/products/$PRODUCT_SLUG"

mkdir -p "$PRODUCT_DIR/docs"
mkdir -p "$PRODUCT_DIR/epics"
mkdir -p "$PRODUCT_DIR/sprints"

# Copy templates
TEMPLATES_DIR="$REPO_ROOT/.specify/templates/bmad"

BRIEF_FILE="$PRODUCT_DIR/docs/product-brief.md"
if [ -f "$TEMPLATES_DIR/product-brief.md" ]; then
    cp "$TEMPLATES_DIR/product-brief.md" "$BRIEF_FILE"
    sed -i "s/\[PRODUCT NAME\]/$PRODUCT_NAME/g" "$BRIEF_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$BRIEF_FILE"
fi

PRD_FILE="$PRODUCT_DIR/docs/prd.md"
if [ -f "$TEMPLATES_DIR/prd-template.md" ]; then
    cp "$TEMPLATES_DIR/prd-template.md" "$PRD_FILE"
    sed -i "s/\[FEATURE NAME\]/$PRODUCT_NAME/g" "$PRD_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$PRD_FILE"
fi

ARCH_FILE="$PRODUCT_DIR/docs/architecture.md"
if [ -f "$TEMPLATES_DIR/architecture.md" ]; then
    cp "$TEMPLATES_DIR/architecture.md" "$ARCH_FILE"
    sed -i "s/\[FEATURE NAME\]/$PRODUCT_NAME/g" "$ARCH_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$ARCH_FILE"
fi

EPICS_FILE="$PRODUCT_DIR/epics/epics.md"
if [ -f "$TEMPLATES_DIR/epics-template.md" ]; then
    cp "$TEMPLATES_DIR/epics-template.md" "$EPICS_FILE"
    sed -i "s/\[FEATURE NAME\]/$PRODUCT_NAME/g" "$EPICS_FILE"
    sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$EPICS_FILE"
fi

if $JSON_MODE; then
    printf '{"PRODUCT_NAME":"%s","PRODUCT_SLUG":"%s","PRODUCT_DIR":"%s","BRIEF_FILE":"%s","PRD_FILE":"%s","ARCH_FILE":"%s","EPICS_FILE":"%s"}\n' \
        "$PRODUCT_NAME" "$PRODUCT_SLUG" "$PRODUCT_DIR" "$BRIEF_FILE" "$PRD_FILE" "$ARCH_FILE" "$EPICS_FILE"
else
    echo "PRODUCT_NAME: $PRODUCT_NAME"
    echo "PRODUCT_SLUG: $PRODUCT_SLUG"
    echo "PRODUCT_DIR: $PRODUCT_DIR"
    echo "BRIEF_FILE: $BRIEF_FILE"
    echo "PRD_FILE: $PRD_FILE"
    echo "ARCH_FILE: $ARCH_FILE"
    echo "EPICS_FILE: $EPICS_FILE"
    echo ""
    echo "Next steps:"
    echo "  1. Fill out the product brief: $BRIEF_FILE"
    echo "  2. Run /bmad.prd to create detailed requirements"
    echo "  3. Run /bmad.architecture to design the system"
fi
