# Installation

## Prerequisites

- Claude Code CLI (`claude`) on PATH
- Go 1.25+ (optional, for building from source)

## Install Wave

### GitHub Release (Recommended)

Download pre-built binaries from [GitHub Releases](https://github.com/re-cinq/wave/releases):

| Platform | Architecture | Archive |
|----------|-------------|---------|
| Linux | x86_64 | `wave_<version>_linux_amd64.tar.gz` |
| Linux | ARM64 | `wave_<version>_linux_arm64.tar.gz` |
| Linux (deb) | x86_64 | `wave_<version>_linux_amd64.deb` |
| Linux (deb) | ARM64 | `wave_<version>_linux_arm64.deb` |
| macOS | Intel | `wave_<version>_darwin_amd64.zip` |
| macOS | Apple Silicon | `wave_<version>_darwin_arm64.zip` |

#### Linux (tar.gz)

```bash
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_linux_amd64.tar.gz
tar -xzf wave_linux_amd64.tar.gz
sudo mv wave /usr/local/bin/
```

#### Linux (deb)

```bash
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_linux_amd64.deb
sudo dpkg -i wave_linux_amd64.deb
```

#### macOS

```bash
# Apple Silicon
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_darwin_arm64.zip
unzip wave_darwin_arm64.zip && sudo mv wave /usr/local/bin/

# Intel
curl -LO https://github.com/re-cinq/wave/releases/latest/download/wave_darwin_amd64.zip
unzip wave_darwin_amd64.zip && sudo mv wave /usr/local/bin/
```

### Build from Source

```bash
git clone https://github.com/re-cinq/wave.git
cd wave
make build
sudo mv wave /usr/local/bin/
```

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

# Debian/Ubuntu
sudo apt remove wave

# Remove project files (optional)
rm -rf ~/.wave
```

## Next Steps

- [Quickstart](/quickstart) - Get running in 60 seconds
- [Pipeline Configuration](/guides/pipeline-configuration) - Configure your first pipeline
