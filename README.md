# Remote NVIM

## Requirements

- [devcontainer cli](https://github.com/devcontainers/cli)

## Usage

```bash
go build -o rnv
chmod +x rnv
./rnv start
```

## Configuration

Settings can be written in `.remote-nvim.yaml`.

example
```bash
port: 12345
dotfilesRepository: https://github.com/utouto97/dotfiles.git
dotfilesTargetPath: ~/.dotfiles
```
