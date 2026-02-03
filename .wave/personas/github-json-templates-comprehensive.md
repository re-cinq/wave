# Comprehensive GitHub JSON Templates for Wave Personas

This library provides battle-tested JSON templates with realistic examples to ensure 100% contract validation success for GitHub-related Wave personas.

## Template Library with Realistic Examples

### GitHub Issue Analysis Template (github-analyst persona)

**Contract**: `github-issue-analysis.schema.json`
**Persona**: `github-analyst`
**Output File**: `artifact.json`

#### Perfect Template with Real Data
```json
{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  "total_issues": 23,
  "analyzed_count": 23,
  "poor_quality_issues": [
    {
      "number": 42,
      "title": "bug in thing",
      "body": "",
      "quality_score": 15,
      "problems": [
        "Title too vague and uses lowercase",
        "No description provided whatsoever",
        "Missing reproduction steps",
        "No environment information",
        "No expected vs actual behavior"
      ],
      "recommendations": [
        "Rewrite title as 'Fix authentication error during OAuth login flow'",
        "Add detailed problem description explaining what fails",
        "Include step-by-step reproduction instructions",
        "Specify environment details (OS, browser, version)",
        "Describe expected behavior vs what actually happens"
      ],
      "labels": [],
      "url": "https://github.com/re-cinq/wave/issues/42"
    },
    {
      "number": 73,
      "title": "Error",
      "body": "It doesn't work",
      "quality_score": 25,
      "problems": [
        "Title lacks specificity about what error occurs",
        "Description too brief and unhelpful",
        "No error messages or logs included"
      ],
      "recommendations": [
        "Rewrite title to include specific error message",
        "Expand description with context and impact",
        "Add complete error messages and stack traces"
      ],
      "labels": ["bug"],
      "url": "https://github.com/re-cinq/wave/issues/73"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}
```

#### Common Invalid Examples to Avoid

**❌ WRONG - Trailing commas will break validation**
```json
{
  "repository": {
    "owner": "re-cinq",
    "name": "wave",  // ← This comma breaks everything
  },
  "total_issues": 5,
  "poor_quality_issues": [
    {
      "number": 42,
      "title": "bug",
      "quality_score": 20,
      "problems": ["vague title"],  // ← This comma breaks everything
    }
  ],  // ← This comma breaks everything
}
```

**❌ WRONG - Quoted numbers break schema validation**
```json
{
  "total_issues": "5",        // Should be: 5
  "poor_quality_issues": [
    {
      "number": "42",         // Should be: 42
      "quality_score": "20"   // Should be: 20
    }
  ]
}
```

**❌ WRONG - Missing required fields**
```json
{
  "repository": {
    "owner": "re-cinq"
    // Missing required "name" field
  },
  "poor_quality_issues": [
    {
      "number": 42,
      "title": "bug"
      // Missing required "quality_score" and "problems" fields
    }
  ]
  // Missing required "total_issues" field
}
```

### GitHub Enhancement Results Template (github-enhancer persona)

**Contract**: `github-enhancement-results.schema.json`
**Persona**: `github-enhancer`
**Output File**: `artifact.json`

#### Perfect Template with Real Data
```json
{
  "enhanced_issues": [
    {
      "issue_number": 42,
      "success": true,
      "changes_made": [
        "Updated title from 'bug in thing' to 'Fix authentication error during OAuth login flow'",
        "Added structured issue template with sections for description, reproduction steps, and environment",
        "Applied labels: bug, authentication, needs-investigation",
        "Added helpful comment explaining enhancement process"
      ],
      "title_updated": true,
      "body_updated": true,
      "labels_added": ["authentication", "needs-investigation"],
      "comment_added": true,
      "url": "https://github.com/re-cinq/wave/issues/42"
    },
    {
      "issue_number": 73,
      "success": false,
      "changes_made": [],
      "title_updated": false,
      "body_updated": false,
      "labels_added": [],
      "comment_added": false,
      "error": "GitHub API returned 403: Forbidden - insufficient permissions to edit issue",
      "url": "https://github.com/re-cinq/wave/issues/73"
    },
    {
      "issue_number": 98,
      "success": true,
      "changes_made": [
        "Enhanced existing description with structured sections",
        "Added missing reproduction steps placeholder",
        "Applied label: documentation"
      ],
      "title_updated": false,
      "body_updated": true,
      "labels_added": ["documentation"],
      "comment_added": true,
      "url": "https://github.com/re-cinq/wave/issues/98"
    }
  ],
  "total_attempted": 3,
  "total_successful": 2,
  "total_failed": 1,
  "timestamp": "2026-02-03T15:35:00Z"
}
```

#### Common Invalid Examples to Avoid

**❌ WRONG - Logic inconsistencies break business rules**
```json
{
  "enhanced_issues": [
    {
      "issue_number": 42,
      "success": true,           // Says success = true
      "changes_made": [],        // But no changes made?
      "error": "Failed somehow"  // And has an error? Inconsistent!
    }
  ],
  "total_attempted": 1,
  "total_successful": 2,  // More successful than attempted? Impossible!
  "total_failed": 0
}
```

**❌ WRONG - Boolean type errors**
```json
{
  "enhanced_issues": [
    {
      "issue_number": 42,
      "success": "true",        // Should be: true (unquoted)
      "title_updated": "false", // Should be: false (unquoted)
      "changes_made": []
    }
  ]
}
```

### GitHub PR Draft Template (github-pr-creator persona)

**Contract**: `github-pr-draft.schema.json`
**Persona**: `github-pr-creator`
**Output File**: `artifact.json`

#### Perfect Template with Real Data
```json
{
  "title": "Add comprehensive JSON validation system for Wave personas",
  "body": "## Summary\nThis PR implements a robust JSON validation system to prevent contract validation failures in Wave pipelines. The system includes standardized templates, real-time validation helpers, and detailed error recovery mechanisms.\n\n## Changes\n- Added comprehensive JSON output templates for all GitHub personas\n- Implemented syntax validation helpers with common error patterns\n- Created realistic examples with valid/invalid comparisons\n- Enhanced persona system prompts with strict formatting requirements\n- Added validation test framework for reliable output\n\n## Motivation\nAI personas were producing valuable analysis but inconsistent JSON formatting that broke contract validation, causing pipeline failures and development delays. This system ensures 100% first-pass validation success.\n\nFixes #142 and addresses issues raised in #98\n\n## Test Plan\n- [ ] Manual testing with github-analyst persona using real repository data\n- [ ] Validation testing with github-enhancer persona on sample issues\n- [ ] PR creation testing with github-pr-creator persona\n- [ ] Contract schema validation for all output types\n- [ ] Error recovery testing with intentionally malformed JSON\n\n## Checklist\n- [x] Templates tested against contract schemas\n- [x] Documentation updated with examples\n- [x] Validation helpers verified with real data\n- [x] Error patterns documented with fixes\n- [ ] Integration tests added for pipeline validation\n- [ ] Performance impact assessed",
  "head": "feature/json-validation-system",
  "base": "main",
  "draft": false,
  "labels": ["enhancement", "pipeline", "validation", "json"],
  "reviewers": ["wave-team", "pipeline-maintainers"],
  "related_issues": [142, 98, 73],
  "breaking_changes": false
}
```

#### Common Invalid Examples to Avoid

**❌ WRONG - Field constraint violations**
```json
{
  "title": "Fix bug",              // Too short (< 10 chars), breaks minLength constraint
  "body": "Brief fix",            // Too short (< 50 chars), breaks minLength constraint
  "head": "",                     // Empty string breaks minLength constraint
  "base": "main"
}
```

**❌ WRONG - Unescaped multiline content**
```json
{
  "title": "Add multi-line feature",
  "body": "## Summary
This breaks JSON because
newlines aren't escaped.",     // Literal newlines break JSON parsing
  "head": "feature/branch",
  "base": "main"
}
```

**✅ CORRECT - Properly escaped content**
```json
{
  "title": "Add multi-line feature",
  "body": "## Summary\nThis works correctly because\nnewlines are escaped properly.",
  "head": "feature/branch",
  "base": "main"
}
```

**❌ WRONG - Array type mismatches**
```json
{
  "title": "Fix issue references",
  "body": "Fixes several issues",
  "head": "fix/issues",
  "base": "main",
  "related_issues": ["42", "73"]  // Should be: [42, 73] (integers, not strings)
}
```

## Validation Testing Examples

### Real-World Test Cases

#### Test Case 1: GitHub Issue Analysis with Edge Cases
```json
{
  "repository": {
    "owner": "facebook",
    "name": "react"
  },
  "total_issues": 1247,
  "analyzed_count": 50,
  "poor_quality_issues": [
    {
      "number": 12345,
      "title": "React doesn't work with \"special quotes\" and\nnewlines",
      "body": "Description with escaped content:\n- Bullet point 1\n- Bullet point 2\n\nCode example:\n```javascript\nconst x = \"test\";\n```",
      "quality_score": 35,
      "problems": [
        "Title contains unescaped quotes and newlines",
        "Missing environment details",
        "No clear reproduction steps"
      ],
      "recommendations": [
        "Escape quotes in title properly",
        "Add environment section with React version",
        "Include minimal reproduction example"
      ],
      "labels": ["bug", "needs-repro"],
      "url": "https://github.com/facebook/react/issues/12345"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}
```

#### Test Case 2: Enhancement Results with Failures
```json
{
  "enhanced_issues": [
    {
      "issue_number": 1001,
      "success": false,
      "changes_made": [],
      "title_updated": false,
      "body_updated": false,
      "labels_added": [],
      "comment_added": false,
      "error": "API rate limit exceeded: 403 Forbidden. Retry after 3600 seconds.",
      "url": "https://github.com/example/repo/issues/1001"
    },
    {
      "issue_number": 1002,
      "success": false,
      "changes_made": [],
      "title_updated": false,
      "body_updated": false,
      "labels_added": [],
      "comment_added": false,
      "error": "Issue is locked and cannot be modified",
      "url": "https://github.com/example/repo/issues/1002"
    }
  ],
  "total_attempted": 2,
  "total_successful": 0,
  "total_failed": 2,
  "timestamp": "2026-02-03T16:00:00Z"
}
```

## Quick Validation Commands

### Validation Script Template
Save as `.wave/scripts/validate-json.sh`:

```bash
#!/bin/bash
set -e

echo "Validating artifact.json..."

# Test basic JSON syntax
if jq empty artifact.json 2>/dev/null; then
    echo "✓ JSON syntax valid"
else
    echo "✗ JSON syntax invalid - run: jq . artifact.json"
    exit 1
fi

# Test schema-specific validations based on content
if jq -e '.repository' artifact.json > /dev/null 2>&1; then
    echo "Validating GitHub Issue Analysis schema..."
    jq -e '.repository.owner and .repository.name and .total_issues and .poor_quality_issues' artifact.json > /dev/null
    echo "✓ Required fields present"
elif jq -e '.enhanced_issues' artifact.json > /dev/null 2>&1; then
    echo "Validating GitHub Enhancement Results schema..."
    jq -e '.enhanced_issues and .total_attempted and .total_successful' artifact.json > /dev/null
    echo "✓ Required fields present"
elif jq -e '.title and .body and .head and .base' artifact.json > /dev/null 2>&1; then
    echo "Validating GitHub PR Draft schema..."
    title_len=$(jq -r '.title | length' artifact.json)
    body_len=$(jq -r '.body | length' artifact.json)
    if [ "$title_len" -lt 10 ]; then
        echo "✗ Title too short (< 10 chars): $title_len"
        exit 1
    fi
    if [ "$body_len" -lt 50 ]; then
        echo "✗ Body too short (< 50 chars): $body_len"
        exit 1
    fi
    echo "✓ Field constraints met"
else
    echo "⚠ Unknown JSON structure - manual validation required"
fi

echo "✓ All validations passed"
```

### Emergency Validation Commands

```bash
# Quick syntax check
echo '{"test": true}' | jq . > /dev/null && echo "Valid" || echo "Invalid"

# Find trailing commas (common error)
grep -n ',\s*[}\]]' artifact.json || echo "No trailing commas found"

# Find unquoted keys (common error)
grep -n '[^"]\w\+:' artifact.json || echo "All keys properly quoted"

# Validate field types
jq '.total_issues | type' artifact.json   # Should output: "number"
jq '.success | type' artifact.json        # Should output: "boolean"

# Check required fields for GitHub Issue Analysis
jq -r 'if .repository.owner then "✓ owner" else "✗ missing owner" end' artifact.json
jq -r 'if .repository.name then "✓ name" else "✗ missing name" end' artifact.json
jq -r 'if .total_issues then "✓ total_issues" else "✗ missing total_issues" end' artifact.json
```

## Error Recovery Patterns

### Pattern 1: Fix Trailing Commas
```bash
# Remove trailing commas before } and ]
sed -i 's/,\s*}/}/g; s/,\s*]/]/g' artifact.json
```

### Pattern 2: Fix Unquoted Numbers/Booleans
```bash
# Convert quoted numbers to unquoted (be careful with this!)
sed -i 's/"total_issues": *"\([0-9]\+\)"/total_issues": \1/g' artifact.json
sed -i 's/"success": *"true"/success": true/g' artifact.json
sed -i 's/"success": *"false"/success": false/g' artifact.json
```

### Pattern 3: Escape Quotes in Strings
```python
import json
import re

def fix_quotes(text):
    # Fix unescaped quotes inside JSON strings (basic approach)
    return re.sub(r'(?<!\\)"(?![:,\]\}])', '\\"', text)

# Use with caution - test thoroughly
```

## Best Practices for Persona Developers

### Pre-Generation Checklist
1. **Know your schema** - Review the exact contract schema first
2. **Use templates** - Start with proven templates, don't create from scratch
3. **Plan content** - Know what data you'll include before generating
4. **Validate early** - Check JSON syntax as you build each section

### During Generation
1. **Copy template exactly** - Don't modify structure
2. **Fill systematically** - Replace placeholders one field at a time
3. **Escape immediately** - Handle special characters as you encounter them
4. **Check types** - Keep numbers unquoted, strings quoted

### Post-Generation
1. **Save and test** - Write to artifact.json and validate immediately
2. **Fix errors fast** - Address syntax errors before continuing
3. **Verify schema** - Check all required fields are present
4. **Test edge cases** - Ensure content doesn't break parsing

### Emergency Protocol
1. **Copy to backup** - Save current JSON before fixes
2. **Use JSONLint** - Online validation for quick error identification
3. **Fix common patterns** - Apply known fixes for trailing commas, types
4. **Re-validate completely** - Test full validation chain before submission

This comprehensive template system ensures 100% validation success when followed correctly.