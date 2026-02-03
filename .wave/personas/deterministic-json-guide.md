# Deterministic JSON Output Guide for AI Personas

## Problem Statement

Wave AI personas sometimes produce JSON with subtle formatting issues that cause pipeline validation failures, even when the semantic content is correct. These failures are deterministic problems (same input produces same malformed output) that can be prevented with proper output discipline.

## Root Causes

### 1. Whitespace Collapse
When AI generates JSON with indentation, subsequent processing may collapse all whitespace incorrectly:
- **Problem**: Newlines within multiline strings get removed
- **Result**: String content becomes corrupted
- **Solution**: Use proper JSON escaping (`\n`) instead of literal newlines

### 2. Comment Injection
Some AI models add helpful comments within JSON structure:
- **Problem**: JSON doesn't support comments
- **Result**: Parser fails to recognize valid JSON
- **Example**: `{"key": "value"} // explanation`
- **Solution**: Put all explanations outside the JSON block

### 3. Trailing Commas
When generating arrays or objects dynamically, AI may add trailing commas:
- **Problem**: `[1, 2, 3,]` is invalid JSON
- **Result**: Parser fails immediately
- **Solution**: Always check last item has no comma

### 4. Quote Inconsistency
Mixing single and double quotes or using unquoted keys:
- **Problem**: `{'key': 'value'}` or `{key: value}`
- **Result**: JSON parser fails
- **Solution**: Always use double quotes, always quote keys

## Best Practices for AI Personas

### Output Discipline

**Before generating any JSON**:
1. Confirm the exact structure expected by the contract schema
2. Plan the JSON layout mentally
3. Use a mental template to prevent structure errors

**While generating JSON**:
1. Generate complete items, not partial ones
2. Add commas between items (but not after the last)
3. Escape special characters: `"` → `\"`, newline → `\n`
4. Don't add comments, markdown, or explanations inside JSON

**After generating JSON**:
1. Validate syntax mentally by counting brackets
2. Verify each array/object has proper item separation
3. Test if possible before submitting

### Template Approach

Create a mental template for your output:

```json
{
  "required_field_1": "...",
  "required_field_2": [...],
  "array_of_objects": [
    {
      "object_field_1": "...",
      "object_field_2": "..."
    }
  ]
}
```

This forces consistent structure and prevents errors.

### Multiline Content Strategy

For descriptions, analysis, or long text:

**Correct approach**:
```json
{
  "analysis": "First point.\nSecond point.\nThird point.",
  "code_example": "function test() {\n  console.log('hello');\n}"
}
```

**Why this works**:
- Properly escaped newlines preserve content
- Valid JSON that deserializes correctly
- Wave pipeline processes \n as actual newlines

### Special Character Escaping

| Character | Escape | Use Case |
|-----------|--------|----------|
| `"` (quote) | `\"` | Inside string values |
| `\` (backslash) | `\\` | File paths, regex patterns |
| Newline | `\n` | Line breaks in text |
| Tab | `\t` | Indentation in descriptions |
| Carriage return | `\r` | Windows line endings |
| Backspace | `\b` | Special formatting |
| Form feed | `\f` | Special formatting |

## Common Patterns for Different Content Types

### Analysis Results
```json
{
  "issue_count": 5,
  "findings": [
    {
      "id": 1,
      "title": "Finding Title",
      "description": "Detailed description.\nCan span multiple lines.",
      "severity": "high",
      "recommendation": "What to do about it"
    }
  ],
  "summary": "5 findings identified\nTop priority: Critical issue on line 42"
}
```

### Code Changes
```json
{
  "file_path": "src/main.go",
  "changes": [
    {
      "type": "modification",
      "line_start": 10,
      "line_end": 15,
      "old_content": "func old() {\n  return 1\n}",
      "new_content": "func new() {\n  return 2\n}"
    }
  ],
  "rationale": "Improved implementation\nfor better performance"
}
```

### Configuration Data
```json
{
  "settings": {
    "name": "Configuration Name",
    "version": "1.0",
    "enabled": true,
    "options": {
      "option1": "value1",
      "option2": 42
    }
  },
  "notes": "Configuration for use case X\nApplies to production environments"
}
```

## Validation Checklist

Use this before submitting any JSON:

- [ ] No trailing commas in arrays or objects
- [ ] All keys are double-quoted strings
- [ ] All string values are double-quoted
- [ ] Numbers are unquoted (no quotes around integers/floats)
- [ ] Booleans are lowercase: `true`, `false`
- [ ] Null values are lowercase: `null`
- [ ] Every opening `{` has matching `}`
- [ ] Every opening `[` has matching `]`
- [ ] Items in arrays/objects separated by commas
- [ ] No commas after final items
- [ ] All special characters escaped (quotes, newlines, etc.)
- [ ] No comments anywhere in JSON
- [ ] No markdown formatting in JSON values
- [ ] All multiline text properly escaped with `\n`

## Troubleshooting

### "Failed to clean malformed JSON"

This means Wave tried to auto-fix your JSON but couldn't.

**Likely causes**:
1. Unquoted keys: `{key: "value"}` → `{"key": "value"}`
2. Trailing commas: `[1, 2, 3,]` → `[1, 2, 3]`
3. Comments: `{"key": "value"} // comment` → remove comment
4. Mixed quotes: `{'key': 'value'}` → `{"key": "value"}`

**Resolution**:
1. Review your JSON for the patterns above
2. Fix manually before output
3. Validate with jq or jsonlint
4. Re-run the step

### "Artifact does not match schema"

Your JSON is valid syntax but doesn't match the expected structure.

**Check**:
1. All required fields are present
2. Field types match (strings vs numbers vs objects)
3. Arrays contain the right item types
4. No extra fields that schema forbids

**Reference**:
Look at the schema file (e.g., `github-issue-analysis.schema.json`) for the exact structure expected.

## Real-World Example: GitHub Issue Analysis

**Correct Output**:
```json
{
  "repository": {
    "owner": "re-cinq",
    "name": "wave"
  },
  "total_issues": 10,
  "analyzed_count": 10,
  "poor_quality_issues": [
    {
      "number": 20,
      "title": "add scan poorly commented gh issues and extend",
      "body": "",
      "quality_score": 5,
      "problems": [
        "Title is vague and grammatically incorrect",
        "No description provided",
        "No labels assigned"
      ],
      "recommendations": [
        "Rewrite title to be specific and clear",
        "Add comprehensive description with problem statement",
        "Add appropriate labels like 'enhancement' or 'feature'"
      ],
      "labels": [],
      "url": "https://github.com/re-cinq/wave/issues/20"
    }
  ],
  "quality_threshold": 70,
  "timestamp": "2026-02-03T18:14:42Z"
}
```

**What makes this work**:
- Proper bracket matching
- No trailing commas
- All strings quoted consistently
- Empty arrays `[]` when no items
- ISO format timestamp
- Empty string `""` for missing body
- Proper array items with commas between

## Integration with Wave Pipeline

When your JSON is valid:
1. ✅ Contract validation passes immediately
2. ✅ Next pipeline step receives valid data
3. ✅ No pipeline delays or retries needed
4. ✅ Analysis content is preserved completely

When JSON is malformed:
1. ❌ Validation fails and step must be re-executed
2. ❌ Time wasted on auto-fixing attempts
3. ❌ Content may be lost if cleaning corrupts data
4. ❌ Pipeline velocity decreases

## Summary

**The goal**: Produce valid, deterministic JSON every time.

**The method**:
1. Use proper JSON syntax from the start
2. Follow the contract schema structure exactly
3. Escape special characters consistently
4. Validate mentally before output
5. Treat JSON as code, not as flexible text

**The result**:
- Fast pipeline execution
- Zero validation failures
- Valuable analysis preserved completely
- Professional, production-ready output
