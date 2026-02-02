# Troubleshooting

Common errors and solutions when working with Wave.

## Binary Not Found

**Error:** `adapter binary 'claude' not found on PATH`

**Solutions:**
```bash
# Install Claude Code
npm install -g @anthropic-ai/claude-code

# Verify installation
which claude

# Add custom location to PATH
export PATH="$PATH:/path/to/bin"
```

---

## Contract Validation Failures

**Error:** `json_schema: required field 'files' missing`

**Solutions:**
1. Check the contract schema: `cat .wave/contracts/output.schema.json`
2. Review step output: `cat .wave/workspaces/<pipeline-id>/<step-id>/output/`
3. Update persona system prompt to match expected format
4. Increase retries:
   ```yaml
   handover:
     contract:
       max_retries: 3
   ```

---

## Timeout Issues

**Error:** `context deadline exceeded`

**Solutions:**
```bash
# Increase timeout via CLI
wave run --pipeline flow.yaml --timeout 60

# Or update manifest
runtime:
  default_timeout_minutes: 45
```

Consider breaking complex tasks into multiple pipeline steps.

---

## Workspace Errors

**Permission denied:**
```bash
mkdir -p /tmp/wave && chmod 755 /tmp/wave
# Or configure different root:
runtime:
  workspace_root: ~/.wave/workspaces
```

**Disk space:**
```bash
wave clean --all --dry-run  # Preview
wave clean --older-than 7d  # Clean old workspaces
```

---

## Persona and Manifest Errors

**Persona not found:**
```yaml
# Add to wave.yaml
personas:
  reviewer:
    adapter: claude
    system_prompt_file: .wave/personas/reviewer.md
```

**System prompt missing:**
```bash
mkdir -p .wave/personas
echo "# Navigator" > .wave/personas/navigator.md
```

**Circular dependency:**
Review `dependencies` arrays and remove cycles. Use `wave validate` to detect issues.

---

## Permission Denied

**Tool blocked by persona:**
```json
{"error":"Write(src/main.go) blocked by persona 'navigator'"}
```

Use the correct persona - navigator is read-only, use craftsman for writes.

**Hook blocked operation:**
Check and fix the hook script:
```bash
.wave/hooks/pre-commit-lint.sh
echo $?  # Should be 0
```

---

## API Errors

**Missing API key:**
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

**Rate limited:** Wave auto-retries. If persistent, reduce `max_concurrent_workers`.

---

## Debugging

```bash
# Enable debug logging
wave run --pipeline flow.yaml --debug 2>debug.log

# Dry run first
wave run --pipeline flow.yaml --dry-run

# Filter events
wave run --pipeline flow.yaml | jq 'select(.state == "failed")'

# Check audit logs
cat .wave/traces/<pipeline-id>.jsonl | jq '.tool_calls'
```
