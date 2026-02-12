# Quickstart

Get your first pipeline running in 60 seconds.

## 1. Install Wave

<InstallTabs />

Verify installation:

```bash
wave --version
```

## 2. Choose Your AI Adapter

Wave executes AI workflows through CLI adapters. Choose one of the supported options:

### Claude Code (Default)

Claude Code is the recommended adapter for Wave, providing the best integration and capabilities.

```bash
# Install Claude Code CLI
npm install -g @anthropic-ai/claude-code

# Verify installation
claude --version
```

::: tip Don't have Node.js?
Install via [nvm](https://github.com/nvm-sh/nvm) (macOS/Linux) or download from [nodejs.org](https://nodejs.org/) (all platforms).
:::

### OpenCode (Alternative)

OpenCode provides an open-source alternative with support for multiple LLM providers.

```bash
# Install OpenCode
go install github.com/opencode-ai/opencode@latest

# Or via Homebrew (macOS/Linux)
brew install opencode

# Verify installation
opencode --version
```

To use OpenCode as your default adapter, configure it in your `wave.yaml`:

```yaml
adapters:
  default: opencode
  opencode:
    model: gpt-4-turbo
```

See [Adapters Reference](/reference/adapters) for complete configuration options.

## 3. Set Your API Key

```bash
export ANTHROPIC_API_KEY="your-api-key"
```

::: tip Don't have an API key?
Get one at [console.anthropic.com](https://console.anthropic.com/). Free tier available for testing.
:::

::: warning API Key Security
Never commit your API key to version control. Add `ANTHROPIC_API_KEY` to your shell profile (`~/.bashrc`, `~/.zshrc`) or use a secrets manager.
:::

## 4. Initialize Your Project

```bash
cd /path/to/your-project
wave init
```

This creates:
- `wave.yaml` - Project manifest
- `.wave/personas/` - AI agent definitions
- `.wave/pipelines/` - Ready-to-run pipelines

::: tip Don't have a codebase?
Wave works great for self-analysis:
```bash
git clone https://github.com/re-cinq/wave.git
cd wave && wave init
```
:::

## 5. Run Your First Pipeline

```bash
wave run hello-world "testing Wave"
```

### Expected Output

With the default TUI mode you'll see a progress bar and spinners. To see text output, use `-o text`:

```bash
wave run hello-world "testing Wave" -o text
```

```
[10:00:01] → greet (craftsman)
[10:00:01]   greet: Executing agent
[10:00:05]   greet: Processing results
[10:00:05] ✓ greet completed (4.0s, 0k tokens)
[10:00:05] → verify (navigator)
[10:00:05]   verify: Executing agent
[10:00:12]   verify: Processing results
[10:00:12] ✓ verify completed (6.9s, 0k tokens)

  ✓ Pipeline 'hello-world' completed successfully (10.9s)

  Deliverables (4):
     • .wave/workspaces/hello-world-aca3e016/greet/CLAUDE.md
     • .wave/workspaces/hello-world-aca3e016/greet/greeting.txt
     • .wave/workspaces/hello-world-aca3e016/verify/CLAUDE.md
     • .wave/workspaces/hello-world-aca3e016/verify/output/result.json
```

### What Just Happened?

1. Wave loaded the `hello-world` pipeline from `.wave/pipelines/`
2. The **greet** step ran with the craftsman persona
3. The **verify** step received the greeting artifact and confirmed it
4. Each step ran with fresh memory (no context bleed between steps)
5. Artifacts were saved to `.wave/workspaces/` for inspection

## Troubleshooting

::: danger ANTHROPIC_API_KEY not set
**Error:** `Error: ANTHROPIC_API_KEY environment variable is not set`

**Solution:** Set your API key before running Wave:
```bash
export ANTHROPIC_API_KEY="sk-ant-..."
```

Or add it permanently to your shell profile:
```bash
echo 'export ANTHROPIC_API_KEY="your-key"' >> ~/.bashrc
source ~/.bashrc
```
:::

::: danger Claude Code not installed
**Error:** `Error: adapter 'claude' not found. Is Claude Code CLI installed?`

**Solution:** Install the Claude Code CLI:
```bash
npm install -g @anthropic-ai/claude-code
```

Verify it's in your PATH:
```bash
which claude  # Should show the installation path
```
:::

::: danger Permission denied errors
**Error:** `Permission denied: cannot write to /usr/local/bin/wave`

**Solution:** Use `sudo` for system-wide installation or install to a user directory:
```bash
# Option 1: Run install script as root (installs to /usr/local/bin)
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sudo sh

# Option 2: Install to user directory
WAVE_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
```
:::

::: warning Common YAML syntax errors
**Error:** `yaml: line X: did not find expected key`

**Common causes and fixes:**

1. **Incorrect indentation** - YAML requires consistent spacing (use 2 spaces, not tabs):
   ```yaml
   # Wrong
   steps:
   	- name: greet  # Tab character

   # Correct
   steps:
     - name: greet  # 2 spaces
   ```

2. **Missing colons or quotes**:
   ```yaml
   # Wrong
   prompt This is a prompt

   # Correct
   prompt: "This is a prompt"
   ```

3. **Invalid special characters** - Wrap strings containing `:`, `#`, or `{` in quotes:
   ```yaml
   # Wrong
   prompt: Review this: analyze the code

   # Correct
   prompt: "Review this: analyze the code"
   ```

**Pro tip:** Validate your YAML with `wave validate` before running pipelines.
:::

## Try a Real Pipeline

Run a code review on your project:

```bash
wave run code-review "review the main module"
```

Or run an ad-hoc task without a pipeline file:

```bash
wave do "analyze the error handling in this codebase"
```

## Quick Commands

```bash
# List available pipelines
wave list pipelines

# Check pipeline status
wave status

# View artifacts from last run
wave artifacts

# View logs
wave logs

# Validate configuration
wave validate

# Clean up workspaces
wave clean
```

## Optional: Enable Sandbox Mode

For isolated development with filesystem and environment protection:

```bash
# Requires Nix (https://nixos.org/download.html)
nix develop
```

This enters a bubblewrap sandbox where the filesystem is read-only except for the project directory, and the home directory (`~/.ssh`, `~/.aws`, etc.) is hidden. See [Sandbox Setup Guide](/guides/sandbox-setup) for details.

## Next Steps

- [Sandbox Setup](/guides/sandbox-setup) - Isolate AI agent sessions
- [Use Cases](/use-cases/) - Find pipelines for code review, documentation, and tests
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure
- [Concepts: Personas](/concepts/personas) - Learn about AI agent roles
- [CLI Reference](/reference/cli) - Complete command documentation
- [Adapters Reference](/reference/adapters) - Configure alternative LLM providers
