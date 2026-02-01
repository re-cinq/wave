# Installation

## Prerequisites

- Go 1.22+ (for building from source)
- Claude Code CLI (`claude`) on PATH
- Git

## Install Muzzle

### Option 1: Pre-built Binary

```bash
# Linux (x64)
curl -L https://github.com/recinq/muzzle/releases/latest/download/muzzle-linux-amd64 -o muzzle
chmod +x muzzle
sudo mv muzzle /usr/local/bin/

# macOS (x64)
curl -L https://github.com/recinq/muzzle/releases/latest/download/muzzle-darwin-amd64 -o muzzle
chmod +x muzzle
sudo mv muzzle /usr/local/bin/

# Verify installation
muzzle --version
```

### Option 2: Build from Source

```bash
git clone https://github.com/recinq/muzzle.git
cd muzzle
go build -o muzzle ./cmd/muzzle/
sudo mv muzzle /usr/local/bin/
```

## Verify Installation

```bash
# Check Muzzle version
muzzle --version

# Check Claude Code is available
claude --version
```

## Next Steps

- [Initialize your first project](/guide/quick-start)
- [Read the configuration guide](/guide/configuration)
- [Browse examples](/examples/)