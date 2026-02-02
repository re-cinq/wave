#!/usr/bin/env bash
# BMAD Quick Spec - Analyze codebase and produce tech-spec with stories

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
            echo "Usage: $0 [--json] <feature_description>"
            echo ""
            echo "BMAD Quick Spec: Analyze codebase and produce tech-spec with stories"
            echo ""
            echo "Options:"
            echo "  --json    Output in JSON format"
            echo "  --help    Show this help message"
            exit 0
            ;;
        *) ARGS+=("$arg") ;;
    esac
done

FEATURE_DESCRIPTION="${ARGS[*]}"
if [ -z "$FEATURE_DESCRIPTION" ]; then
    echo "Usage: $0 [--json] <feature_description>" >&2
    exit 1
fi

REPO_ROOT=$(get_repo_root)
cd "$REPO_ROOT"

# Create bmad directory structure
BMAD_DIR="$REPO_ROOT/.bmad"
mkdir -p "$BMAD_DIR/specs"
mkdir -p "$BMAD_DIR/sprints"

# Generate spec ID based on timestamp
SPEC_ID=$(date +%Y%m%d-%H%M%S)
SPEC_DIR="$BMAD_DIR/specs/$SPEC_ID"
mkdir -p "$SPEC_DIR"

# Create spec file from template
TEMPLATE="$REPO_ROOT/.specify/templates/bmad/story-template.md"
SPEC_FILE="$SPEC_DIR/quick-spec.md"

# Initialize with basic structure
cat > "$SPEC_FILE" << 'TEMPLATE_EOF'
# Quick Tech Spec: [FEATURE]

**ID**: [SPEC_ID]
**Created**: [DATE]
**Status**: Draft

## Feature Description

[DESCRIPTION]

## Codebase Analysis

### Relevant Files
[To be filled by analysis]

### Existing Patterns
[To be filled by analysis]

### Integration Points
[To be filled by analysis]

## Technical Approach

### Changes Required
[To be filled after analysis]

### Dependencies
[To be filled after analysis]

## Stories

### Story 1: [Title]
**Points**: [TBD]
**Priority**: P0

**Description**:
[To be filled]

**Acceptance Criteria**:
- [ ] [Criterion 1]
- [ ] [Criterion 2]

---

## Review Checklist
- [ ] Codebase analyzed
- [ ] Patterns identified
- [ ] Stories defined
- [ ] Dependencies identified
- [ ] Ready for implementation
TEMPLATE_EOF

# Replace placeholders
sed -i "s/\[SPEC_ID\]/$SPEC_ID/g" "$SPEC_FILE"
sed -i "s/\[DATE\]/$(date +%Y-%m-%d)/g" "$SPEC_FILE"
sed -i "s/\[DESCRIPTION\]/$FEATURE_DESCRIPTION/g" "$SPEC_FILE"
sed -i "s/\[FEATURE\]/$FEATURE_DESCRIPTION/g" "$SPEC_FILE"

if $JSON_MODE; then
    printf '{"SPEC_ID":"%s","SPEC_DIR":"%s","SPEC_FILE":"%s","BMAD_DIR":"%s"}\n' \
        "$SPEC_ID" "$SPEC_DIR" "$SPEC_FILE" "$BMAD_DIR"
else
    echo "SPEC_ID: $SPEC_ID"
    echo "SPEC_DIR: $SPEC_DIR"
    echo "SPEC_FILE: $SPEC_FILE"
    echo "BMAD_DIR: $BMAD_DIR"
fi
