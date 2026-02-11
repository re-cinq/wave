# Installation

## Prerequisites

- Claude Code CLI (`claude`) on PATH
- Git (optional, for building from source)
- Go 1.25+ (optional, for building from source)

## Install Wave

### Install Script (Recommended)

The install script detects your OS and architecture, downloads the appropriate binary from GitHub Releases, verifies the SHA256 checksum, and installs it.

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh -s -- 0.1.0
```

Override the install directory:

```bash
WAVE_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

::: warning Private Repository
The curl one-liner only works when the repository is **public**. While the repository is private, clone the repo and run `scripts/install.sh` directly.
:::

### Homebrew (macOS)

```bash
brew install re-cinq/tap/wave
```

### Debian / Ubuntu

Download the `.deb` package from [GitHub Releases](https://github.com/re-cinq/wave/releases):

```bash
# Download the latest .deb package
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_<VERSION>_linux_amd64.deb

# Install
sudo dpkg -i wave_*.deb
```

### Arch Linux (AUR)

```bash
# Using an AUR helper (e.g., yay)
yay -S wave

# Or manually
git clone https://aur.archlinux.org/wave.git
cd wave
makepkg -si
```

### Nix / NixOS

```bash
# Install directly from the flake
nix profile install github:re-cinq/wave

# Or run without installing
nix run github:re-cinq/wave -- --help
```

### Manual Download

Download pre-built archives from [GitHub Releases](https://github.com/re-cinq/wave/releases):

| Platform | Architecture | Archive |
|----------|-------------|---------|
| Linux | x86_64 | `wave_VERSION_linux_amd64.tar.gz` |
| Linux | ARM64 | `wave_VERSION_linux_arm64.tar.gz` |
| macOS | Intel | `wave_VERSION_darwin_amd64.zip` |
| macOS | Apple Silicon | `wave_VERSION_darwin_arm64.zip` |

```bash
# Example: Linux x86_64
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_VERSION_linux_amd64.tar.gz
tar -xzf wave_*.tar.gz
sudo mv wave /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
make build
sudo mv wave /usr/local/bin/
```

## Verify Installation

```bash
# Check version and build info
wave --version

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

# Homebrew
brew uninstall wave

# Debian/Ubuntu
sudo apt remove wave

# Nix
nix profile remove wave

# Remove project files (optional)
rm -rf ~/.wave
```

## Next Steps

- [Initialize your first project](/guide/quick-start)
- [Read the configuration guide](/guide/configuration)
- [Browse examples](/examples/)
