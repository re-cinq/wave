# Error Code Reference

Complete catalog of Wave error codes with descriptions, common causes, and resolution steps.

## Error Code Format

Wave error codes follow the format `WAVE_EXXX` where `XXX` is a three-digit number. Errors are categorized by range:

| Range | Category |
|-------|----------|
| E001-E099 | Configuration errors |
| E100-E199 | Execution errors |
| E200-E299 | Validation errors |
| E300-E399 | Resource errors |

## Configuration Errors

### WAVE_E001: Manifest Not Found

**Description**: Wave could not locate a `wave.yaml` manifest file in the project.

**Error Message**:
```
WAVE_E001: manifest not found: no wave.yaml in current directory or parent directories
```

**Common Causes**:
- Running Wave outside a Wave-initialized project
- The `wave.yaml` file was deleted or renamed
- Working directory is incorrect

**Resolution Steps**:

1. Initialize a new Wave project:
   ```bash
   wave init
   ```

2. Verify you're in the correct directory:
   ```bash
   pwd
   ls wave.yaml
   ```

3. Check parent directories for existing manifest:
   ```bash
   find . -name "wave.yaml" -type f 2>/dev/null
   ```

4. Create a minimal manifest manually:
   ```yaml
   apiVersion: v1
   kind: WaveManifest
   metadata:
     name: my-project
   adapters:
     claude:
       binary: claude
       mode: headless
   personas:
     navigator:
       adapter: claude
       system_prompt_file: .wave/personas/navigator.md
   runtime:
     workspace_root: /tmp/wave
   ```

---

### WAVE_E002: Invalid YAML Syntax

**Description**: The manifest file contains invalid YAML syntax that cannot be parsed.

**Error Message**:
```
WAVE_E002: invalid YAML syntax in wave.yaml: line 15: mapping values are not allowed here
```

**Common Causes**:
- Incorrect indentation (YAML uses spaces, not tabs)
- Missing colons after keys
- Unquoted special characters
- Malformed strings or arrays

**Resolution Steps**:

1. Validate YAML syntax using an online validator or local tool:
   ```bash
   yamllint wave.yaml
   ```

2. Check for common issues:
   - Use spaces, not tabs (2 spaces per indentation level)
   - Quote strings containing special characters (`:`, `#`, `@`, etc.)
   - Ensure consistent indentation throughout

3. Use Wave's built-in validation:
   ```bash
   wave validate --verbose
   ```

4. Example of correct YAML formatting:
   ```yaml
   # Correct
   metadata:
     name: my-project
     description: "Project with special chars: #@!"

   # Incorrect - will cause E002
   metadata:
   name: my-project    # Wrong indentation
     description: Project with special chars: #@!  # Unquoted special chars
   ```

---

### WAVE_E003: Unknown Persona Reference

**Description**: A pipeline step references a persona that is not defined in the manifest.

**Error Message**:
```
WAVE_E003: unknown persona reference: persona 'reviewer' not found in manifest
```

**Common Causes**:
- Typo in persona name in pipeline definition
- Persona definition missing from `wave.yaml`
- Case sensitivity mismatch

**Resolution Steps**:

1. List available personas:
   ```bash
   wave list personas
   ```

2. Check the pipeline file for typos:
   ```yaml
   steps:
     - id: review
       persona: reviewer  # Is this defined in wave.yaml?
   ```

3. Add the missing persona to `wave.yaml`:
   ```yaml
   personas:
     reviewer:
       adapter: claude
       description: "Code review specialist"
       system_prompt_file: .wave/personas/reviewer.md
   ```

4. Verify persona name case matches exactly (YAML is case-sensitive):
   ```yaml
   # These are different personas:
   # - Navigator
   # - navigator
   # - NAVIGATOR
   ```

---

## Execution Errors

### WAVE_E004: Contract Validation Failed

**Description**: The output from a pipeline step did not match the expected contract schema.

**Error Message**:
```
WAVE_E004: contract validation failed: required field 'files' missing in output
```

**Common Causes**:
- AI output doesn't match expected JSON schema
- Missing required fields in output
- Type mismatches (string vs array, etc.)
- Output format changed after prompt modification

**Resolution Steps**:

1. Review the contract definition:
   ```bash
   cat .wave/contracts/output.schema.json
   ```

2. Check the actual step output:
   ```bash
   cat .wave/workspaces/<pipeline-id>/<step-id>/output/result.json
   ```

3. Update the persona system prompt to specify the exact output format:
   ```markdown
   ## Output Format

   You must output valid JSON with this structure:
   {
     "files": ["file1.go", "file2.go"],
     "analysis": "Your analysis here",
     "score": 85
   }
   ```

4. Increase contract retry attempts:
   ```yaml
   steps:
     - id: analyze
       handover:
         contract:
           max_retries: 3
   ```

5. Make the contract more lenient if appropriate:
   ```json
   {
     "type": "object",
     "properties": {
       "files": { "type": "array" }
     },
     "required": []  // Remove required fields
   }
   ```

---

### WAVE_E005: Adapter Not Available

**Description**: The configured adapter binary could not be found or executed.

**Error Message**:
```
WAVE_E005: adapter not available: binary 'claude' not found on PATH
```

**Common Causes**:
- Adapter CLI not installed
- Binary not on system PATH
- Incorrect binary name in configuration
- Permission issues with binary

**Resolution Steps**:

1. Install the required adapter:
   ```bash
   # For Claude Code
   npm install -g @anthropic-ai/claude-code

   # Verify installation
   which claude
   claude --version
   ```

2. Add binary location to PATH:
   ```bash
   # In ~/.bashrc or ~/.zshrc
   export PATH="$PATH:$HOME/.local/bin"
   ```

3. Check the binary name in `wave.yaml`:
   ```yaml
   adapters:
     claude:
       binary: claude  # Must match actual binary name
       mode: headless
   ```

4. Verify binary permissions:
   ```bash
   ls -la $(which claude)
   chmod +x $(which claude)
   ```

5. In CI/CD, ensure adapter is installed before running Wave:
   ```yaml
   - npm install -g @anthropic-ai/claude-code
   - wave run code-review
   ```

---

### WAVE_E006: Permission Denied

**Description**: An operation was blocked by persona permissions or system security.

**Error Message**:
```
WAVE_E006: permission denied: Write(src/main.go) blocked by persona 'navigator'
```

**Common Causes**:
- Persona lacks required tool permissions
- Operation matches a deny pattern
- Hook script rejected the operation
- Filesystem permission issue

**Resolution Steps**:

1. Check persona permissions in `wave.yaml`:
   ```yaml
   personas:
     navigator:
       permissions:
         allowed_tools: ["Read", "Glob", "Grep"]
         deny: ["Write(*)", "Edit(*)"]  # This blocks writes
   ```

2. Use the correct persona for the operation:
   - `navigator` - Read-only analysis
   - `craftsman` - Implementation with write access
   - `auditor` - Security review (read-only)

3. Update permissions if needed:
   ```yaml
   personas:
     navigator:
       permissions:
         allowed_tools: ["Read", "Write(docs/*)"]  # Allow writing to docs/
         deny: ["Write(src/*)"]
   ```

4. Check hook scripts:
   ```bash
   # Review pre-tool hooks
   cat .wave/hooks/pre-commit-lint.sh

   # Ensure hook exits 0 for allowed operations
   echo $?
   ```

5. For filesystem permissions:
   ```bash
   # Check directory permissions
   ls -la /path/to/directory

   # Fix ownership if needed
   chmod 755 /path/to/directory
   ```

---

### WAVE_E007: Step Timeout Exceeded

**Description**: A pipeline step exceeded its configured timeout duration.

**Error Message**:
```
WAVE_E007: step timeout exceeded: step 'analyze' exceeded 30m timeout
```

**Common Causes**:
- Complex analysis taking longer than expected
- LLM API latency issues
- Network connectivity problems
- Insufficient timeout configuration

**Resolution Steps**:

1. Increase step timeout:
   ```bash
   wave run pipeline --timeout 60
   ```

2. Configure timeout in `wave.yaml`:
   ```yaml
   runtime:
     default_timeout_minutes: 45
   ```

3. Set per-step timeout in pipeline:
   ```yaml
   steps:
     - id: analyze
       timeout_minutes: 60
   ```

4. Break complex steps into smaller operations:
   ```yaml
   steps:
     - id: analyze-part1
       persona: navigator
       exec:
         source: "Analyze authentication module"

     - id: analyze-part2
       persona: navigator
       dependencies: [analyze-part1]
       exec:
         source: "Analyze database module"
   ```

5. Check for network issues:
   ```bash
   # Test API connectivity
   curl -I https://api.anthropic.com

   # Check DNS resolution
   nslookup api.anthropic.com
   ```

---

### WAVE_E008: Context Limit Reached

**Description**: The conversation context exceeded the LLM's token limit.

**Error Message**:
```
WAVE_E008: context limit reached: conversation exceeded 200000 tokens
```

**Common Causes**:
- Large codebase analysis without filtering
- Accumulated context from many operations
- Large file contents in prompts
- Relay/compaction not triggered

**Resolution Steps**:

1. Enable context relay:
   ```yaml
   runtime:
     relay:
       token_threshold_percent: 80
       strategy: summarize_to_checkpoint
   ```

2. Reduce scope of analysis:
   ```bash
   wave run analyze --input "Focus on src/auth/ directory only"
   ```

3. Use file filtering in prompts:
   ```yaml
   steps:
     - id: analyze
       exec:
         source: |
           Analyze only these files:
           - src/main.go
           - src/handler.go
           Ignore test files and vendor/.
   ```

4. Break into multiple pipeline runs:
   ```bash
   # Run for each module separately
   wave run analyze --input "Module: auth"
   wave run analyze --input "Module: api"
   wave run analyze --input "Module: db"
   ```

5. Configure meta-pipeline limits:
   ```yaml
   runtime:
     meta_pipeline:
       max_total_tokens: 500000
   ```

---

## Validation Errors

### WAVE_E200: Schema Validation Failed

**Description**: The manifest or pipeline YAML does not conform to the expected schema.

**Error Message**:
```
WAVE_E200: schema validation failed: field 'runtime.workspace_root' must be a string
```

**Resolution Steps**:

1. Run validation:
   ```bash
   wave validate --verbose
   ```

2. Check field types against schema documentation
3. Review [Manifest Schema Reference](/reference/manifest-schema)

---

### WAVE_E201: Circular Dependency Detected

**Description**: Pipeline steps form a circular dependency chain.

**Error Message**:
```
WAVE_E201: circular dependency detected: step1 -> step2 -> step3 -> step1
```

**Resolution Steps**:

1. Review step dependencies in pipeline
2. Remove or restructure circular references
3. Use `wave validate` to detect cycles

---

## Resource Errors

### WAVE_E300: Workspace Creation Failed

**Description**: Wave could not create the ephemeral workspace directory.

**Error Message**:
```
WAVE_E300: workspace creation failed: permission denied creating /tmp/wave
```

**Resolution Steps**:

1. Check workspace root permissions:
   ```bash
   mkdir -p /tmp/wave && chmod 755 /tmp/wave
   ```

2. Configure alternative workspace:
   ```yaml
   runtime:
     workspace_root: ~/.wave/workspaces
   ```

---

### WAVE_E301: Disk Space Exhausted

**Description**: Insufficient disk space for workspace operations.

**Error Message**:
```
WAVE_E301: disk space exhausted: workspace requires 500MB, only 100MB available
```

**Resolution Steps**:

1. Clean old workspaces:
   ```bash
   wave clean --older-than 7d
   ```

2. Check available space:
   ```bash
   df -h /tmp/wave
   ```

3. Use different workspace location with more space

---

## Quick Reference

| Code | Name | Category | Quick Fix |
|------|------|----------|-----------|
| WAVE_E001 | Manifest Not Found | Config | Run `wave init` |
| WAVE_E002 | Invalid YAML Syntax | Config | Check YAML formatting |
| WAVE_E003 | Unknown Persona | Config | Add persona to manifest |
| WAVE_E004 | Contract Failed | Execution | Review contract schema |
| WAVE_E005 | Adapter Not Available | Execution | Install adapter binary |
| WAVE_E006 | Permission Denied | Execution | Check persona permissions |
| WAVE_E007 | Step Timeout | Execution | Increase `--timeout` |
| WAVE_E008 | Context Limit | Execution | Enable relay compaction |

## Debug Mode

For detailed error information, run Wave with debug logging:

```bash
wave run pipeline --debug 2>debug.log

# Filter specific error codes
grep "WAVE_E" debug.log
```

## Getting Help

If you encounter an error not listed here:

1. Check the [Troubleshooting Guide](/reference/troubleshooting)
2. Search [GitHub Issues](https://github.com/re-cinq/wave/issues)
3. Open a new issue with:
   - Error code and full message
   - Wave version (`wave --version`)
   - Relevant manifest snippets
   - Debug log output

## See Also

- [Troubleshooting Reference](/reference/troubleshooting)
- [Manifest Schema](/reference/manifest-schema)
- [CI/CD Integration](/guides/ci-cd)
