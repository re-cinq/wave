# Installation

## Prerequisites

- Claude Code CLI (`claude`) on PATH
- Git (optional, for building from source)
- Go 1.22+ (optional, for building from source)

## Install Wave

### Quick Install (Recommended)

Clone and run the install script:

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
./install.sh
```

The script automatically detects your OS and architecture, downloads the appropriate binary, and installs it to your PATH.

#### Install Options

```bash
# Install specific version
WAVE_VERSION=v1.0.0 ./install.sh

# Custom install directory
WAVE_INSTALL_DIR=~/bin ./install.sh

# Skip PATH modification
WAVE_NO_MODIFY_PATH=1 ./install.sh
```

#### One-liner Install

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/install.sh | sh
```

::: warning Private Repository
The curl one-liner only works when the repository is **public**. While the repository is private or internal, use the git clone method above.
:::

### Manual Download

Download pre-built binaries directly:

```bash
# Linux (x64)
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# Linux (ARM64)
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-linux-arm64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-darwin-amd64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/

# macOS (Apple Silicon)
curl -L https://github.com/re-cinq/wave/releases/latest/download/wave-darwin-arm64 -o wave
chmod +x wave
sudo mv wave /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/re-cinq/wave.git
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