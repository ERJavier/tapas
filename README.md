# TAPAS

A modern, keyboard-driven TUI for monitoring and managing listening ports. Think *htop for ports*: small, sharp, terminal-native.

**Platforms:** macOS and Linux only.

## Install

```bash
go install github.com/javiercepeda/tapas@latest
```

Or clone and build:

```bash
git clone https://github.com/javiercepeda/tapas
cd tapas && go build -o tapas .
./tapas
```

## Usage

| Key | Action |
|-----|--------|
| `↑` / `↓` / `j` / `k` | Navigate |
| `Enter` | Details (port, PID, process, command, working dir) |
| `k` | Kill selected port (with confirmation) |
| `r` | Refresh list |
| `q` | Quit |

## Requirements

- **macOS:** `lsof`, `ps` (default)
- **Linux:** `ss`, `/proc` (default)

## Roadmap

- **v0.1 (MVP)** – List ports, navigate, kill, refresh, quit *(current)*
- **v0.2** – Sort, filter, color semantics, polish
- **v1.0** – Framework badges, Docker detection, optional watch mode

## License

MIT
