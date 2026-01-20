# Installation

## From Releases (Recommended)

Download the latest binary from the [releases page](https://github.com/arch-err/autogitter/releases).

### Linux

```bash
# amd64
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-linux-amd64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/

# arm64
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-linux-arm64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/
```

### macOS

```bash
# Apple Silicon (M1/M2/M3)
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-darwin-arm64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/

# Intel
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-darwin-amd64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/
```

### Windows

Download `ag-windows-amd64.exe` from the [releases page](https://github.com/arch-err/autogitter/releases) and add it to your PATH.

```powershell
# PowerShell - download to current directory
Invoke-WebRequest -Uri "https://github.com/arch-err/autogitter/releases/latest/download/ag-windows-amd64.exe" -OutFile "ag.exe"
```

## From Source

Requires Go 1.24 or later.

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

Make sure `$GOPATH/bin` (or `$HOME/go/bin`) is in your PATH.

## From Git

Clone and build manually:

```bash
git clone https://github.com/arch-err/autogitter.git
cd autogitter
go build -o ag ./cmd/ag
sudo mv ag /usr/local/bin/
```

Or using the Taskfile (if you have [Task](https://taskfile.dev/) installed):

```bash
git clone https://github.com/arch-err/autogitter.git
cd autogitter
task build
task install
```

## Verify Installation

```bash
ag --version
```

## Shell Completions

Generate shell completions for your shell:

```bash
# Bash
ag completion bash > /etc/bash_completion.d/ag

# Zsh
ag completion zsh > "${fpath[1]}/_ag"

# Fish
ag completion fish > ~/.config/fish/completions/ag.fish
```

## Updating

### From Releases

Download the new binary and replace the old one:

```bash
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-linux-amd64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/
```

### From Source

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

## Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/ag

# Remove config (optional)
rm -rf ~/.config/autogitter

# Remove credentials (optional)
rm -rf ~/.local/share/autogitter
```
