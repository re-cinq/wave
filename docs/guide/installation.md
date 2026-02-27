# Installation

## Prerequisites

- Claude Code CLI (`claude`) on PATH
- Go 1.25+ (optional, for building from source)

## Install Wave

### Build from Source

Build from source:

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
make build
sudo mv wave /usr/local/bin/
```

Or install to a user directory without sudo:

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
make build
mkdir -p ~/.local/bin
mv wave ~/.local/bin/
```

### Install Script

The install script detects your OS and architecture, downloads the correct binary from [GitHub Releases](https://github.com/re-cinq/wave/releases), and verifies the SHA256 checksum:

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh -s -- 0.3.0
```

Override the install directory:

```bash
WAVE_INSTALL_DIR=~/bin curl -fsSL https://raw.githubusercontent.com/re-cinq/wave/main/scripts/install.sh | sh
```

**Supported platforms:** Linux (x86_64, ARM64), macOS (Intel, Apple Silicon)

The script installs to `/usr/local/bin` when run as root, or `~/.local/bin` otherwise. It warns you if the install directory is not on your `PATH`.

### Debian / Ubuntu (.deb)

```bash
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_linux_amd64.deb
sudo dpkg -i wave_linux_amd64.deb
```

### Manual Download

Download pre-built archives from [GitHub Releases](https://github.com/re-cinq/wave/releases):

| Platform | Architecture | Archive |
|----------|-------------|---------|
| Linux | x86_64 | `wave_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `wave_<version>_linux_arm64.tar.gz` |
| macOS | Intel | `wave_<version>_darwin_amd64.zip` |
| macOS | Apple Silicon | `wave_<version>_darwin_arm64.zip` |

## Versioning

Wave follows [Semantic Versioning](https://semver.org/). Releases are created automatically on every merge to `main`, with the version bump determined by [conventional commit](https://www.conventionalcommits.org/) prefixes:

- `fix:`, `docs:`, `chore:` -> **patch** (0.0.X)
- `feat:` -> **minor** (0.X.0)
- `feat!:` or `BREAKING CHANGE:` -> **major** (X.0.0)

## Verify Installation

```bash
wave --version
wave --help
claude --version
```

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/wave
# or
rm ~/.local/bin/wave

# Debian/Ubuntu
sudo apt remove wave

# Remove project files (optional)
rm -rf ~/.wave
```

## Next Steps

- [Quickstart](/quickstart) - Get running in 60 seconds
- [Pipeline Configuration](/guides/pipeline-configuration) - Configure your first pipeline
