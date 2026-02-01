# Installation

## Prerequisites

- Go 1.22+ (for building from source)
- Claude Code CLI (`claude`) on PATH
- Git

## Install Wave

### Option 1: Pre-built Binary

```bash
# Linux (x64)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# macOS (x64)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-darwin-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# Verify installation
wave --version
```

### Option 2: Build from Source

```bash
git clone https://github.com/recinq/wave.git
cd wave
go build -o wave ./cmd/wave/
sudo mv wave /usr/local/bin/
```

## Verify Installation

```bash
# Check Wave version
wave --version

# Check Claude Code is available
claude --version
```

## Next Steps

- [Initialize your first project](/guide/quick-start)
- [Read the configuration guide](/guide/configuration)
- [Browse examples](/examples/)