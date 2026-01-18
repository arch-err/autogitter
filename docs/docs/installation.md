# Installation

## From Source

Requires Go 1.21 or later.

```bash
go install github.com/arch-err/autogitter/cmd/ag@latest
```

## From Releases

Download the latest binary from the [releases page](https://github.com/arch-err/autogitter/releases).

```bash
# Linux (amd64)
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-linux-amd64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/

# macOS (arm64)
curl -L https://github.com/arch-err/autogitter/releases/latest/download/ag-darwin-arm64 -o ag
chmod +x ag
sudo mv ag /usr/local/bin/
```

## From Git

```bash
git clone https://github.com/arch-err/autogitter.git
cd autogitter
task build
task install
```
