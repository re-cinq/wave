# Installation

## Prerequisites

- Claude Code CLI (`claude`) on PATH
- Git (optional, for building from source)
- Go 1.22+ (optional, for building from source)

## Install Wave

### Quick Install (Recommended)

The install script automatically detects your OS and architecture:

```bash
curl -fsSL https://raw.githubusercontent.com/recinq/wave/main/install.sh | sh
```

This will:
- Detect your platform (Linux, macOS, Windows via WSL)
- Download the appropriate binary
- Install to `/usr/local/bin` (or `~/.local/bin` if no sudo)
- Add to PATH if needed

#### Install Options

```bash
# Install specific version
WAVE_VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/recinq/wave/main/install.sh | sh

# Custom install directory
WAVE_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/recinq/wave/main/install.sh | sh

# Skip PATH modification
WAVE_NO_MODIFY_PATH=1 curl -fsSL https://raw.githubusercontent.com/recinq/wave/main/install.sh | sh
```

### Manual Download

Download pre-built binaries directly:

```bash
# Linux (x64)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-linux-arm64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-darwin-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/recinq/wave/releases/latest/download/wave-darwin-arm64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/recinq/wave.git
cd wave
go build -o wave ./cmd/wave/
sudo mv wave /usr/local/bin/
```

## Verify Installation

```bash
# Check Wave is installed
wave --help

# Check Claude Code is available
claude --version
```

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/wave
# or
rm ~/.local/bin/wave

# Remove project files (optional)
rm -rf ~/.wave
```

## Next Steps

- [Initialize your first project](/guide/quick-start)
- [Read the configuration guide](/guide/configuration)
- [Browse examples](/examples/)