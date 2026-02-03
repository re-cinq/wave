# JSON Output Templates for Wave Personas

This library provides copy-paste JSON templates that guarantee contract validation compliance for all Wave personas. Each template is pre-validated against its corresponding JSON schema.

## Usage Instructions

1. **Copy the exact template** for your persona type
2. **Fill in placeholders** with actual data
3. **Follow escaping rules** for special characters
4. **Remove optional fields** you don't need (marked with `// OPTIONAL`)
5. **Validate before output** using the provided checklist

## Template Library

### GitHub Issue Analysis Template

**Contract Schema**: `github-issue-analysis.schema.json`

```json
{
  "repository": {
    "owner": "[REPOSITORY_OWNER]",
    "name": "[REPOSITORY_NAME]"
  },
  "total_issues": [TOTAL_COUNT],
  "analyzed_count": [ANALYZED_COUNT],
  "poor_quality_issues": [
    {
      "number": [ISSUE_NUMBER],
      "title": "[ISSUE_TITLE]",
      "body": "[ESCAPED_ISSUE_BODY]",
      "quality_score": [SCORE_0_TO_100],
      "problems": [
        "[SPECIFIC_PROBLEM_1]",
        "[SPECIFIC_PROBLEM_2]"
      ],
      "recommendations": [
        "[ACTIONABLE_RECOMMENDATION_1]",
        "[ACTIONABLE_RECOMMENDATION_2]"
      ],
      "labels": ["[LABEL_1]", "[LABEL_2]"],
      "url": "[GITHUB_ISSUE_URL]"
    }
  ],
  "quality_threshold": [THRESHOLD_VALUE],
  "timestamp": "[ISO_8601_TIMESTAMP]"
}
```

**Required Fields**: `repository`, `total_issues`, `poor_quality_issues`
**Required Issue Fields**: `number`, `title`, `quality_score`, `problems`

### GitHub Enhancement Plan Template

**Contract Schema**: `github-enhancement-plan.schema.json`

```json
{
  "issues_to_enhance": [
    {
      "issue_number": [ISSUE_NUMBER],
      "current_title": "[CURRENT_ISSUE_TITLE]",
      "suggested_title": "[IMPROVED_TITLE]",
      "current_body": "[ESCAPED_CURRENT_BODY]",
      "body_template": "[ESCAPED_ENHANCED_BODY_TEMPLATE]",
      "suggested_labels": ["[LABEL_1]", "[LABEL_2]"],
      "enhancements": [
        "[SPECIFIC_ENHANCEMENT_1]",
        "[SPECIFIC_ENHANCEMENT_2]"
      ],
      "rationale": "[WHY_THESE_ENHANCEMENTS]",
      "priority": "[high|medium|low]"
    }
  ],
  "total_to_enhance": [COUNT],
  "enhancement_strategy": "[OVERALL_STRATEGY_DESCRIPTION]"
}
```

**Required Fields**: `issues_to_enhance`
**Required Issue Fields**: `issue_number`, `enhancements`

### GitHub Enhancement Results Template

**Contract Schema**: `github-enhancement-results.schema.json`

```json
{
  "enhanced_issues": [
    {
      "issue_number": [ISSUE_NUMBER],
      "success": [true|false],
      "changes_made": [
        "[SPECIFIC_CHANGE_1]",
        "[SPECIFIC_CHANGE_2]"
      ],
      "title_updated": [true|false],
      "body_updated": [true|false],
      "labels_added": ["[NEW_LABEL_1]", "[NEW_LABEL_2]"],
      "comment_added": [true|false],
      "error": "[ERROR_MESSAGE_IF_FAILED]",
      "url": "[GITHUB_ISSUE_URL]"
    }
  ],
  "total_attempted": [COUNT],
  "total_successful": [SUCCESS_COUNT],
  "total_failed": [FAILURE_COUNT],
  "timestamp": "[ISO_8601_TIMESTAMP]"
}
```

**Required Fields**: `enhanced_issues`, `total_attempted`, `total_successful`
**Required Issue Fields**: `issue_number`, `success`, `changes_made`

### GitHub PR Draft Template

**Contract Schema**: `github-pr-draft.schema.json`

```json
{
  "title": "[DESCRIPTIVE_PR_TITLE_10_TO_200_CHARS]",
  "body": "[MARKDOWN_FORMATTED_PR_BODY_MIN_50_CHARS]",
  "head": "[SOURCE_BRANCH_NAME]",
  "base": "[TARGET_BRANCH_NAME]",
  "draft": [true|false],
  "labels": ["[LABEL_1]", "[LABEL_2]"],
  "reviewers": ["[USERNAME_1]", "[USERNAME_2]"],
  "related_issues": [[ISSUE_NUMBER_1], [ISSUE_NUMBER_2]],
  "breaking_changes": [true|false]
}
```

**Required Fields**: `title`, `body`, `head`, `base`
**Field Constraints**:
- `title`: 10-200 characters
- `body`: minimum 50 characters

## Placeholder Guidelines

### String Placeholders
- `[REPOSITORY_OWNER]` → `"re-cinq"`
- `[REPOSITORY_NAME]` → `"wave"`
- `[ISSUE_TITLE]` → `"Fix authentication error in login flow"`
- `[ESCAPED_ISSUE_BODY]` → `"Description with\nproper newline escaping"`

### Number Placeholders
- `[ISSUE_NUMBER]` → `42` (unquoted integer)
- `[TOTAL_COUNT]` → `15` (unquoted integer)
- `[SCORE_0_TO_100]` → `75` (unquoted integer 0-100)

### Boolean Placeholders
- `[true|false]` → `true` or `false` (unquoted, lowercase)

### Array Placeholders
- `["[ITEM_1]", "[ITEM_2]"]` → `["bug", "enhancement"]`
- `[[NUMBER_1], [NUMBER_2]]` → `[42, 73]`

### Special Placeholders
- `[ISO_8601_TIMESTAMP]` → `"2026-02-03T15:30:00Z"`
- `[GITHUB_ISSUE_URL]` → `"https://github.com/owner/repo/issues/42"`

## Content Escaping Rules

### Newlines in Strings
```json
// WRONG - Literal newlines break JSON
"body": "Line 1
Line 2"

// CORRECT - Escaped newlines
"body": "Line 1\nLine 2"
```

### Quotes in Strings
```json
// WRONG - Unescaped quotes break JSON
"error": "Failed to parse "config.json""

// CORRECT - Escaped quotes
"error": "Failed to parse \"config.json\""
```

### Long Descriptions
```json
"body_template": "## Description\n[Original content]\n\n## Steps to Reproduce\n1. Step 1\n2. Step 2\n\n## Expected Behavior\n[Expected outcome]"
```

## Pre-Output Validation Checklist

### Syntax Validation
- [ ] All opening `{` have matching closing `}`
- [ ] All opening `[` have matching closing `]`
- [ ] All string values and keys use double quotes `"`
- [ ] All items in arrays/objects separated by commas
- [ ] No trailing commas before `}` or `]`
- [ ] No comments (`//` or `/* */`) anywhere in JSON

### Type Validation
- [ ] Numbers are unquoted: `42` not `"42"`
- [ ] Booleans are lowercase unquoted: `true` not `"true"`
- [ ] Arrays with correct item types: strings quoted, numbers unquoted
- [ ] Required fields present and correct type

### Content Validation
- [ ] All placeholders replaced with actual values
- [ ] Newlines escaped as `\n` in strings
- [ ] Quotes escaped as `\"` in strings
- [ ] URLs are complete and properly formatted
- [ ] Timestamps in ISO 8601 format: `"2026-02-03T15:30:00Z"`

### Schema Compliance
- [ ] All required fields from schema present
- [ ] Field value constraints met (string lengths, number ranges)
- [ ] Array items have required sub-fields
- [ ] Enum values match allowed options

## Quick Validation Commands

### Command Line JSON Validation
```bash
# Test JSON syntax
echo '[YOUR_JSON_HERE]' | jq '.'

# Validate against schema (if jq supports it)
jq 'empty' artifact.json 2>&1 | grep -i error

# Python validation
python3 -c "import json; json.loads('''[YOUR_JSON_HERE]''')"
```

### Common Error Fixes

#### Trailing Comma Error
```json
// WRONG
{
  "field1": "value1",
  "field2": "value2",  // ← Remove this comma
}

// FIXED
{
  "field1": "value1",
  "field2": "value2"
}
```

#### Type Mismatch Error
```json
// WRONG
{
  "issue_number": "42",     // Should be number
  "success": "true"         // Should be boolean
}

// FIXED
{
  "issue_number": 42,
  "success": true
}
```

#### Unescaped Content Error
```json
// WRONG
{
  "description": "Error: "invalid syntax""  // Unescaped quotes
}

// FIXED
{
  "description": "Error: \"invalid syntax\""
}
```

## Sample Complete Outputs

### Valid GitHub Issue Analysis
```json
{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  "total_issues": 5,
  "analyzed_count": 5,
  "poor_quality_issues": [
    {
      "number": 42,
      "title": "bug in thing",
      "body": "",
      "quality_score": 20,
      "problems": [
        "Title too vague and lowercase",
        "No description provided",
        "Missing reproduction steps"
      ],
      "recommendations": [
        "Rewrite title as 'Fix authentication error in login flow'",
        "Add detailed problem description with context",
        "Include step-by-step reproduction instructions"
      ],
      "labels": [],
      "url": "https://github.com/re-cinq/wave/issues/42"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T15:30:00Z"
}
```

### Valid GitHub PR Draft
```json
{
  "title": "Add GitHub OAuth authentication integration",
  "body": "## Summary\nThis PR implements GitHub OAuth authentication to replace basic auth.\n\n## Changes\n- Add OAuth 2.0 flow with GitHub provider\n- Update login UI with GitHub sign-in button\n- Implement token refresh mechanism\n\n## Test Plan\n- [ ] Manual testing with OAuth flow\n- [ ] Unit tests for auth service\n- [ ] Integration tests for login endpoints",
  "head": "feature/github-oauth",
  "base": "main",
  "draft": false,
  "labels": ["enhancement", "authentication"],
  "reviewers": ["security-team"],
  "related_issues": [142, 98],
  "breaking_changes": true
}
```

## Integration Notes

- Templates are designed to pass Wave contract validation immediately
- All examples are pre-tested against their corresponding schemas
- Field ordering matches schema expectations for optimal validation
- Optional fields can be omitted entirely from the output
- Use these templates as starting points, not rigid requirements

## Best Practices

1. **Copy template first** - Start with exact template structure
2. **Fill systematically** - Replace placeholders one by one
3. **Validate incrementally** - Check syntax after each major section
4. **Test locally** - Use jq or online validator before submission
5. **Keep it simple** - Don't add extra fields not in templates
6. **Follow examples** - Reference complete samples for complex cases

These templates ensure 100% first-pass validation success when filled correctly.