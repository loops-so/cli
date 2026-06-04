# Loops CLI

Manage your [Loops](https://loops.so) account from the terminal — send transactional emails, manage contacts, events, mailing lists, and more.

> [!NOTE]
> This is beta software. While we're cooking up our first official release there may be breaking changes and other bugs.

## Usage

```
loops [command] [flags]
```

Run `loops --help` to see available commands, or `loops [command] --help` for details on a specific command.

See [the docs](https://loops.so/docs/cli) for comprehensive usage.

## Installation

### Homebrew

```
brew install loops-so/tap/loops
```

### Script for macOS, Linux, Windows via WSL

```bash
curl -fsSL https://cli.loops.so | bash
```

To install a specific version or to a custom path, append `-s -- <version> <path>` to `bash` in the command above. The default installation path is `~/.local/bin`.

### Script for Windows PowerShell

```
irm https://raw.githubusercontent.com/Loops-so/cli/main/install.ps1 | iex
```

### Go install

```bash
go install github.com/loops-so/cli/cmd/loops@latest
```

## Local development

Build the local CLI into `dist/loops`, then put that directory first on `PATH`
so local smoke tests use the same command shape as production:

```bash
task build
export PATH="$PWD/dist:$PATH"
export LOOPS_ENDPOINT_URL=http://localhost:3000/api/v1
loops --output json api-key
loops --output json campaigns create --name "Local API demo"
loops --output json workflows list
loops --output json workflows get <workflow_id>
loops --output json transactional list
```

## Auth

The CLI requires a Loops API key. Get one from [Settings > API](https://app.loops.so/settings?page=api).

### Keyring storage

Store a key with `loops auth login --name <name>`. Run this again with a different name to store keys for multiple teams.

- `loops auth use <name>` — set a stored key as the default
- `loops auth list` — list stored keys and see which is the default

Use `--team <name>` on any command to pick a specific stored key.

### Precedence

When multiple keys are available, the CLI resolves them in this order:

1. `LOOPS_API_KEY` env var
1. `--team` flag
1. The current default (set via `loops auth use`)

## Environment variables

| Variable | Description |
| --- | --- |
| `LOOPS_API_KEY` | API key to use directly — useful for CI or when keyring storage isn't available. |
| `LOOPS_ENDPOINT_URL` | API base URL. Defaults to `https://app.loops.so/api/v1`; set this to a local server for development. |
| `NO_COLOR` | Set to `1` to disable color output. `0` or unset leaves color on. |
