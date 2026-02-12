# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Product Vision

TAPAS is **"a modern, beautiful htop for ports"** - a terminal-native tool for monitoring and managing listening ports. Small, sharp, keyboard-driven, and delightful.

**Target platforms:** macOS and Linux only (no Windows support in scope).

## Architecture

### Core Separation Principle

Keep system logic and UI logic strictly separated:

- **`internal/ports/`** - Port/process detection, lifecycle management, OS command execution
- **`internal/ui/`** - Rendering, input handling, terminal interactions

**Critical rule:** UI code must not execute OS commands directly.

### Core Data Model

The stable core model centers around port metadata:
- Port number
- PID
- Process name
- Protocol
- Uptime/start time
- Working directory

## Development Phases

Build in strict sequence - do not jump phases:

1. **v0.1 MVP** - List listening ports, navigate, kill, refresh, quit
2. **v0.2 UX** - Sort, filter, detail/confirm modals, color semantics
3. **v1.0 Smart Detection** - Framework badges, Docker detection, watch mode

Do not implement phase-3 features until MVP stability is solid on macOS and Linux.

## UX Requirements

### Keyboard-First Interaction

Every core action must be keyboard-accessible:

- `↑`/`↓`/`→` - Navigate
- `Enter` - Inspect details
- `k` - Kill selected process/port (with confirmation)
- `r` - Refresh
- `/` - Search/filter
- `s` - Sort
- `q` - Quit (and close modals)

Avoid multi-key combinations unless absolutely necessary.

### Performance Standards

- Refresh must feel instant
- Never block UI during data updates
- Avoid flicker and full redraws
- Show loading indicator only if refresh exceeds ~300ms
- Manual refresh by default (no continuous polling)

### Safety and Confirmation

Destructive actions must:
- Require explicit confirmation
- Clearly state consequences
- Never be silent

Example: `Kill port 3000 (node)? [y] Confirm [n] Cancel`

### Color Usage

Color communicates state, not decoration:
- Long-running ports (>24h): soft red
- Common dev ports (3000-3005): blue
- Database ports (5432, 6379): purple
- System processes: muted

Color cannot be the only signal - always include text/symbols for accessibility.

## Hard Non-Goals

Reject or defer changes that introduce:
- Cloud integration
- Telemetry
- Background daemons
- Auto-kill scripts
- Windows-specific implementation

## Decision Filter

When implementing features, prefer options that:
1. Improve developer workflow clarity
2. Keep TAPAS lightweight
3. Maintain clean architectural separations

**If a feature does not improve clarity or developer workflow, it does not belong.**

## Error Handling

Errors must be:
- Visible
- Understandable
- Non-fatal to UI responsiveness

Handle common cases clearly: permission denied, command parsing failed, no ports found.

## Tone

- Serious developer tool
- Slightly playful, mostly professional
- No emojis in app UI copy
