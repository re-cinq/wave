# Quickstart

Get your first pipeline running in 60 seconds.

## 1. Install Wave

<script setup>
import PlatformTabs from './.vitepress/theme/components/PlatformTabs.vue'

const installTabs = [
  {
    platform: 'macos',
    label: 'macOS',
    content: `
<div class="install-option">
<h4>Homebrew <span class="recommended-badge">Recommended</span></h4>
<p>Install Wave using the Homebrew package manager:</p>
<pre><code>brew install wave</code></pre>
</div>

<div class="install-option">
<h4>Binary Download</h4>
<p>Download the latest release for macOS:</p>
<pre><code># Intel Mac
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-darwin-amd64 -o /usr/local/bin/wave
chmod +x /usr/local/bin/wave

# Apple Silicon (M1/M2/M3)
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-darwin-arm64 -o /usr/local/bin/wave
chmod +x /usr/local/bin/wave</code></pre>
</div>
`
  },
  {
    platform: 'linux',
    label: 'Linux',
    content: `
<div class="install-option">
<h4>APT (Debian/Ubuntu) <span class="recommended-badge">Recommended</span></h4>
<p>Add the Wave repository and install:</p>
<pre><code># Add Wave repository
curl -fsSL https://packages.wave.dev/gpg | sudo gpg --dearmor -o /usr/share/keyrings/wave-archive-keyring.gpg
echo "deb [signed-by=/usr/share/keyrings/wave-archive-keyring.gpg] https://packages.wave.dev/apt stable main" | sudo tee /etc/apt/sources.list.d/wave.list

# Install Wave
sudo apt update
sudo apt install wave</code></pre>
</div>

<div class="install-option">
<h4>Binary Download</h4>
<p>Download the latest release for Linux:</p>
<pre><code># x86_64
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-amd64 -o /usr/local/bin/wave
chmod +x /usr/local/bin/wave

# ARM64
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-arm64 -o /usr/local/bin/wave
chmod +x /usr/local/bin/wave</code></pre>
</div>
`
  },
  {
    platform: 'windows',
    label: 'Windows',
    content: `
<div class="install-option">
<h4>Scoop <span class="recommended-badge">Recommended</span></h4>
<p>Install Wave using Scoop package manager:</p>
<pre><code>scoop bucket add wave https://github.com/re-cinq/scoop-bucket
scoop install wave</code></pre>
</div>

<div class="install-option">
<h4>Chocolatey</h4>
<p>Install Wave using Chocolatey:</p>
<pre><code>choco install wave</code></pre>
</div>

<div class="install-option">
<h4>Binary Download</h4>
<p>Download the latest release for Windows:</p>
<pre><code># PowerShell (run as Administrator)
Invoke-WebRequest -Uri "https://github.com/re-cinq/wave/releases/latest/download/wave-windows-amd64.exe" -OutFile "C:\\Program Files\\wave\\wave.exe"

# Add to PATH
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\\Program Files\\wave", "Machine")</code></pre>
</div>
`
  }
]
</script>

<PlatformTabs :tabs="installTabs" />

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

You should see progress output similar to this:

```
[10:00:01] started   greet   (craftsman)                 Starting step
[10:00:15] completed greet   (craftsman)   14s    1.2k   Step completed
[10:00:16] started   verify  (navigator)                 Starting step
[10:00:28] completed verify  (navigator)   12s    0.8k   Step completed

Pipeline hello-world completed in 26s
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
# Option 1: Use sudo
sudo curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-amd64 -o /usr/local/bin/wave
sudo chmod +x /usr/local/bin/wave

# Option 2: Install to user directory
mkdir -p ~/.local/bin
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-amd64 -o ~/.local/bin/wave
chmod +x ~/.local/bin/wave
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

## Next Steps

- [Use Cases](/use-cases/) - Find pipelines for code review, security audits, docs, and tests
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure
- [Concepts: Personas](/concepts/personas) - Learn about AI agent roles
- [CLI Reference](/reference/cli) - Complete command documentation
- [Adapters Reference](/reference/adapters) - Configure alternative LLM providers
