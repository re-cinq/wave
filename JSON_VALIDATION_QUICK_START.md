# JSON Validation - Quick Start Guide

## The Problem (Fixed)

Wave JSON validation was failing on valid analysis because the cleaner function collapsed all whitespace, breaking multiline strings.

**Before**: `"description": "Line 1\nLine 2"` → `"description": "Line 1 Line 2"` ❌

**After**: Properly preserved ✓

## What Changed

1. **JSON cleaner** no longer destroys newlines in strings
2. **Earlier validation** checks if JSON is already valid before cleaning
3. **Smarter regex** for comment removal won't mangle string content
4. **New JSONCleaner utility** helps personas produce valid JSON

## For AI Personas

If you output JSON, follow this checklist:

```
Before submitting JSON output:
[ ] No trailing commas: {a:1,} ❌ → {a:1} ✓
[ ] Double quotes only: {'a':'1'} ❌ → {"a":"1"} ✓
[ ] All keys quoted: {a:1} ❌ → {"a":1} ✓
[ ] Newlines escaped: literal newline ❌ → \n ✓
[ ] No comments: // or /* */ ❌ → removed ✓
[ ] Proper escapes: " → \", \ → \\
[ ] Brackets match: Count { matches } matches [ matches ]
[ ] Numbers unquoted: "123" ❌ → 123 ✓
[ ] Booleans lowercase: true/false (not True/False)
[ ] Test with jq: jq . artifact.json ✓
```

## Common Fixes

### Trailing Commas
```json
// Wrong
{
  "items": [1, 2, 3,]
}

// Right
{
  "items": [1, 2, 3]
}
```

### Quote Inconsistency
```json
// Wrong
{'name': 'value'}

// Right
{"name": "value"}
```

### Multiline Text
```json
// Wrong (literal newlines break JSON)
{
  "description": "Line 1
Line 2"
}

// Right (escaped newlines work)
{
  "description": "Line 1\nLine 2"
}
```

### Comments
```json
// Wrong
{
  "value": 123  // This is a comment
}

// Right
{
  "value": 123
}
```

## Testing Your JSON

### Using jq
```bash
jq . artifact.json
# If no error: JSON is valid ✓
# If error: shows what's wrong ❌
```

### Using Python
```bash
python3 -c "import json; json.load(open('artifact.json'))"
# If no error: JSON is valid ✓
```

### Using Online Tool
Visit https://jsonlint.com and paste your JSON

## Files to Reference

### For Personas
- `.wave/personas/github-analyst.md` - GitHub issue analyzer requirements
- `.wave/personas/deterministic-json-guide.md` - Comprehensive JSON best practices
- `.wave/personas/json-output-helper.md` - Quick lookup table

### For Developers
- `internal/contract/json_cleaner.go` - JSONCleaner utility code
- `internal/contract/jsonschema.go` - Enhanced validation logic
- `JSON_VALIDATION_FIX.md` - Full technical details

## Why This Matters

Valid JSON = faster pipelines = better results

When JSON is malformed:
- ❌ Validation fails
- ❌ Step must re-run
- ❌ Content may be lost
- ❌ Pipeline slower

When JSON is valid from the start:
- ✓ Validation passes immediately
- ✓ Content preserved perfectly
- ✓ Next step processes data
- ✓ Pipeline runs at full speed

## Quick Integration

If you're using JSONCleaner in code:

```go
import "github.com/recinq/wave/internal/contract"

cleaner := &contract.JSONCleaner{}

// Check if valid
if cleaner.IsValidJSON(myJSON) {
    // Use it
}

// Clean if needed
cleaned, changes, err := cleaner.CleanJSONOutput(myJSON)
if err != nil {
    // Still can't parse
}

// Format nicely
formatted, _ := cleaner.ValidateAndFormatJSON(myJSON)

// Extract JSON from text
json, err := cleaner.ExtractJSONFromText(myText)
```

## When Things Go Wrong

### Error: "failed to clean malformed JSON"

Your JSON is too broken to fix automatically.

**Check**:
1. All `{` have matching `}`
2. All `[` have matching `]`
3. No unquoted keys
4. No trailing commas

### Error: "artifact does not match schema"

Your JSON is valid but doesn't match expected structure.

**Check**:
1. All required fields present
2. Field types correct (strings/numbers/objects)
3. Correct number of array items
4. No extra unexpected fields

### Verify Locally First

Before submitting to pipeline:

```bash
# Check syntax
jq . artifact.json > /dev/null && echo "Valid JSON"

# Check against schema
jq '. as $data | input as $schema | $schema | . as $s | $data | ... ' \
  artifact.json schema.json
```

## Summary

1. Write deterministic JSON with clear structure
2. Use proper escaping and quoting
3. Test with jq before submitting
4. Reference the guides when unsure
5. Let Wave's validation confirm success

**Result**: Fast, reliable, content-preserving JSON processing ✓
