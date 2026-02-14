# TAPAS

A modern, keyboard-driven TUI for monitoring and managing listening ports. Think *htop for ports*: small, sharp, terminal-native.

**Platforms:** macOS and Linux only.

## Install

**From anywhere** (requires Go):

```bash
go install github.com/javiercepeda/tapas@latest
```

**From the repo** (after clone):

```bash
make install
# or
./install.sh
```

Then run `tapas` from any directory. The binary is installed to `$GOBIN` or `$GOPATH/bin`; ensure that directory is on your `PATH`.

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
