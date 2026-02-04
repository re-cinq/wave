# Quickstart

Get your first pipeline running in 60 seconds.

## 1. Install Wave

```bash
# Linux
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave && sudo mv wave /usr/local/bin/

# macOS
curl -L https://github.com/recinq/wave/releases/latest/download/wave-darwin-amd64 -o wave
chmod +x wave && sudo mv wave /usr/local/bin/
```

Verify installation:

```bash
wave --version
```

## 2. Install Claude CLI

Wave runs AI workflows through Claude Code CLI.

```bash
npm install -g @anthropic-ai/claude-code
claude --version
```

> **Don't have Node.js?** Install via [nvm](https://github.com/nvm-sh/nvm) or download from [nodejs.org](https://nodejs.org/).

> **Don't have Claude CLI?** Wave supports other adapters. See [Adapters Reference](/reference/adapters) to configure alternatives like OpenCode.

## 3. Set Your API Key

```bash
export ANTHROPIC_API_KEY="your-api-key"
```

> **Don't have an API key?** Get one at [console.anthropic.com](https://console.anthropic.com/). Free tier available for testing.

## 4. Initialize Your Project

```bash
cd your-project
wave init
```

This creates:
- `wave.yaml` - Project manifest
- `.wave/personas/` - AI agent definitions
- `.wave/pipelines/` - Ready-to-run pipelines

> **Don't have a codebase?** Wave works great for self-analysis:
> ```bash
> git clone https://github.com/recinq/wave.git
> cd wave && wave init
> ```

## 5. Run Your First Pipeline

```bash
wave run hello-world "testing Wave"
```

Expected output:

```
[10:00:01] started   greet   (craftsman)                 Starting step
[10:00:15] completed greet   (craftsman)   14s    1.2k   Step completed
[10:00:16] started   verify  (navigator)                 Starting step
[10:00:28] completed verify  (navigator)   12s    0.8k   Step completed

Pipeline hello-world completed in 26s
```

## What Just Happened?

1. Wave loaded the `hello-world` pipeline from `.wave/pipelines/`
2. The **greet** step ran with the craftsman persona
3. The **verify** step received the greeting artifact and confirmed it
4. Each step ran with fresh memory (no context bleed between steps)
5. Artifacts were saved to `.wave/workspaces/` for inspection

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

# Clean up workspaces
wave clean
```

## Next Steps

- [Use Cases](/use-cases/) - Find pipelines for code review, security audits, docs, and tests
- [Concepts: Pipelines](/concepts/pipelines) - Understand pipeline structure
- [CLI Reference](/reference/cli) - Complete command documentation
