# Quickstart

Get your first pipeline running in 60 seconds.

## 1. Install Wave

<InstallTabs />

Verify installation:

```bash
wave --version
```

## 2. Install an Adapter

Wave runs every step through an AI coding CLI. `wave init` supports two:

| Adapter | Default model (all tiers) | Install |
|---------|---------------------------|---------|
| Claude Code (`claude`) | `sonnet` | `npm install -g @anthropic-ai/claude-code` |
| Codex (`codex`) | `gpt-6.2-sol` | `npm install -g @openai/codex` or `brew install --cask codex` |

```bash
# Claude Code
npm install -g @anthropic-ai/claude-code
claude --version

# or Codex
npm install -g @openai/codex
codex --version
```

::: tip Don't have Node.js?
Install via [nvm](https://github.com/nvm-sh/nvm) (macOS/Linux) or download from [nodejs.org](https://nodejs.org/) (all platforms).
:::

::: info Other adapters
Wave also ships OpenCode and Gemini adapters, but `wave init` does not set them up and local models are not recommended for real runs yet. See [Adapters Reference](/reference/adapters) to add one to `wave.yaml` by hand.
:::

## 3. Sign In

Start the CLI once and sign in:

- **Claude Code:** run `claude` and log in, or export `ANTHROPIC_API_KEY`.
- **Codex:** run `codex` and choose "Sign in with ChatGPT", or set up an API key.

::: warning API Key Security
Never commit API keys to version control. Keep them in your shell profile or a secrets manager.
:::

## 4. Initialize Your Project

```bash
cd /path/to/your-project
wave init
```

In a terminal, `wave init` asks:

1. Whether to create a git repository, if there is none (Wave runs each step in a git worktree)
2. The adapter: `claude` or `codex` (the first one found on `PATH` is the default)
3. The model, pinned for every complexity tier (`sonnet` for claude, `gpt-6.2-sol` for codex)
4. Any test, lint or build command it could not detect from your project files

Pass `--yes` to accept every default, or answer through flags:

```bash
wave init --yes --adapter codex
wave init --adapter claude --model sonnet --test-command "make test"
```

This creates:
- `wave.yaml` - Project manifest (one adapter, project commands, metadata)
- `.agents/personas/`, `.agents/pipelines/`, `.agents/contracts/`, `.agents/prompts/` - Ready-to-run defaults
- `AGENTS.md` - Project instructions copied into every agent workspace (only if missing)
- `CLAUDE.md` - Imports `AGENTS.md` for Claude Code (claude adapter, only if missing)
- `.gitignore` entries for Wave's runtime state (`.agents/workspaces/`, `.agents/state.db*`, ...)

To change adapter or model later, keeping your customisations:

```bash
wave init --reconfigure --adapter codex --model gpt-6.2-sol
```

Check the setup:

```bash
wave doctor
```

::: tip Don't have a codebase?
Wave works great for self-analysis:
```bash
git clone https://github.com/re-cinq/wave.git
cd wave && wave init
```
:::

## 5. Run Your First Pipeline

```bash
wave run ops-hello-world "testing Wave"
```

### Expected Output

By default you see a progress display. To see plain text output, use `-o text`:

```bash
wave run ops-hello-world "testing Wave" -o text
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

  ✓ Pipeline 'ops-hello-world' completed successfully (10.9s)

  Deliverables (4):
     • .agents/workspaces/ops-hello-world-aca3e016/greet/CLAUDE.md
     • .agents/workspaces/ops-hello-world-aca3e016/greet/greeting.txt
     • .agents/workspaces/ops-hello-world-aca3e016/verify/CLAUDE.md
     • .agents/workspaces/ops-hello-world-aca3e016/verify/output/result.json
```

### What Just Happened?

1. Wave loaded the `ops-hello-world` pipeline from `.agents/pipelines/`
2. The **greet** step ran with the craftsman persona
3. The **verify** step received the greeting artifact and confirmed it
4. Each step ran with fresh memory (no context bleed between steps)
5. Artifacts were saved to `.agents/workspaces/` for inspection

## Troubleshooting

::: danger Adapter not installed
**Error:** `adapter binary not found on PATH: claude`

**Solution:** `wave run` checks the adapter before any step starts. Install the adapter CLI (see step 2), run with the other adapter (`--adapter codex`), or switch the project:
```bash
wave init --reconfigure --adapter codex
```
:::

::: danger Not signed in
**Symptom:** the first step fails with an authentication error from the adapter.

**Solution:** run `claude` or `codex` once and sign in (see step 3).
:::

::: danger Permission denied errors
**Error:** `Permission denied: cannot write to /usr/local/bin/wave`

**Solution:** Build from source and install to a user directory:
```bash
# Option 1: Install to /usr/local/bin (requires sudo)
git clone https://github.com/re-cinq/wave.git
cd wave && make build
sudo mv wave /usr/local/bin/

# Option 2: Install to user directory
git clone https://github.com/re-cinq/wave.git
cd wave && make build
mkdir -p ~/.local/bin && mv wave ~/.local/bin/
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
wave run ops-pr-review "review the main module"
```

Or generate a pipeline for a task none of the shipped ones fit:

```bash
wave meta "analyze the error handling in this codebase" --dry-run
```

## Quick Commands

```bash
# List available pipelines
wave list pipelines

# Check pipeline status (running pipelines; --all for recent runs)
wave status --all

# View artifacts from last run
wave artifacts

# View logs
wave logs

# Validate configuration and check adapters, git and tools
wave validate
wave doctor

# Clean up workspaces
wave clean
```

## Optional: Enable Sandbox Mode

For isolated development with filesystem and environment protection:

```bash
# Requires Nix (https://nixos.org/download.html)
nix develop
```

This enters a bubblewrap sandbox where the filesystem is read-only except for the project directory, and the home directory (`~/.ssh`, `~/.aws`, etc.) is hidden. See [Sandbox Setup Guide](/guide/sandbox-setup) for details.

## Next Steps

- [Sandbox Setup](/guide/sandbox-setup) - Isolate AI agent sessions
- Use Cases - Find pipelines for code review, documentation, and tests
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure
- [Concepts: Personas](/concepts/personas) - Learn about AI agent roles
- [CLI Reference](/reference/cli) - Complete command documentation
- [Adapters Reference](/reference/adapters) - Configure alternative LLM providers
