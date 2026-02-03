# JSON Quick Reference Card for Wave Personas

**CRITICAL**: Use this for every JSON output to ensure 100% pipeline success.

## 📋 30-Second Validation Checklist

### Before You Start (5 seconds)
- [ ] Know your persona type and required schema
- [ ] Have the correct template from `json-output-templates.md`

### While You Generate (2 minutes)
- [ ] Copy exact template structure
- [ ] Replace ALL `[PLACEHOLDERS]` with real data
- [ ] Use double quotes `"` for all strings and keys
- [ ] Keep numbers and booleans unquoted: `42`, `true`, `false`

### Before You Submit (30 seconds)
- [ ] **Bracket Balance**: Count `{` = `}`, `[` = `]`
- [ ] **No Trailing Commas**: Remove `,` before `}` or `]`
- [ ] **Quote Consistency**: All `"double quotes"`, no `'single quotes'`
- [ ] **Required Fields**: All mandatory fields present

## 🚨 Critical Error Patterns

| ❌ WRONG | ✅ CORRECT | Impact |
|----------|------------|--------|
| `{"key": "value",}` | `{"key": "value"}` | Pipeline Failure |
| `{'key': 'value'}` | `{"key": "value"}` | Parse Error |
| `{"count": "42"}` | `{"count": 42}` | Schema Violation |
| `{"success": "true"}` | `{"success": true}` | Type Mismatch |

## 📐 Templates by Persona

### GitHub Issue Analyst
```json
{
  "repository": {"owner": "STRING", "name": "STRING"},
  "total_issues": NUMBER,
  "poor_quality_issues": [
    {
      "number": NUMBER,
      "title": "STRING",
      "quality_score": NUMBER_0_TO_100,
      "problems": ["STRING_ARRAY"],
      "recommendations": ["STRING_ARRAY"]
    }
  ]
}
```

### GitHub Issue Enhancer
```json
{
  "enhanced_issues": [
    {
      "issue_number": NUMBER,
      "success": BOOLEAN,
      "changes_made": ["STRING_ARRAY"]
    }
  ],
  "total_attempted": NUMBER,
  "total_successful": NUMBER,
  "total_failed": NUMBER
}
```

### GitHub PR Creator
```json
{
  "title": "STRING_10_TO_200_CHARS",
  "body": "STRING_MIN_50_CHARS",
  "head": "BRANCH_NAME",
  "base": "BRANCH_NAME",
  "draft": BOOLEAN,
  "related_issues": [NUMBER_ARRAY]
}
```

## 🔧 Instant Fixes

### Trailing Comma Fix
```
Find: "value",}
Replace: "value"}

Find: "value",]
Replace: "value"]
```

### Quote Fix
```
Find: 'text'
Replace: "text"

Find: {key:
Replace: {"key":
```

### Type Fix
```
Find: "42"     (numbers)
Replace: 42

Find: "true"   (booleans)
Replace: true

Find: "false"  (booleans)
Replace: false
```

## 🎯 Schema Requirements

### GitHub Issue Analysis
- **Required**: `repository`, `total_issues`, `poor_quality_issues`
- **Types**: `repository` (object), `total_issues` (number ≥0), `poor_quality_issues` (array)
- **Issue items need**: `number` (int), `title` (string), `quality_score` (0-100), `problems` (array)

### GitHub Enhancement Results
- **Required**: `enhanced_issues`, `total_attempted`, `total_successful`
- **Logic**: `total_attempted = total_successful + total_failed`
- **Issue items need**: `issue_number` (int), `success` (bool), `changes_made` (array)

### GitHub PR Draft
- **Required**: `title`, `body`, `head`, `base`
- **Constraints**: `title` 10-200 chars, `body` min 50 chars
- **Types**: All strings except `related_issues` (number array), `draft` (bool), `breaking_changes` (bool)

## ⚡ Emergency Protocol

### If JSON Validation Fails:
1. **Copy to JSONLint.com** - paste and click validate
2. **Read error message** - shows exactly what's wrong
3. **Apply common fixes**:
   - Remove trailing commas
   - Add quotes around unquoted keys
   - Fix bracket imbalance
4. **Re-validate** until clean
5. **Submit corrected version**

### Common Error Messages:
- `Unexpected token ','` → Remove trailing comma
- `Expected property name` → Add quotes around object key
- `Unexpected end of JSON` → Missing closing bracket
- `Unexpected token '}'` → Missing comma between items

## 🎮 Test Commands

### Quick Syntax Test
```bash
echo 'YOUR_JSON_HERE' | jq '.'
# If pretty-prints → Valid ✓
# If error → Invalid ❌
```

### Field Check
```bash
jq '.repository.owner' artifact.json    # Should return string
jq '.total_issues' artifact.json        # Should return number
jq '.enhanced_issues | length' artifact.json  # Should return count
```

## 🏆 Success Pattern

```
1. Copy Template → 2. Fill Data → 3. Validate → 4. Submit
      📋             ✏️          ✅         📤
   (5 seconds)   (2 minutes)  (30 seconds)  (Success!)
```

**Remember**: Perfect JSON = Smooth Pipeline = Happy Developer 🚀

---
**Quick Access**: `json-output-templates.md` | `json-syntax-validator.md` | `json-format-enforcement.md`