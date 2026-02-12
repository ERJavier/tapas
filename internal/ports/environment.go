package ports

import (
	"path/filepath"
	"strings"
)

// DetectEnvironment returns how the process was launched (npm, yarn, pnpm, poetry, pipenv, cargo, go).
// Empty string if not detected. Uses first token of command (supports paths like /usr/bin/npm).
func DetectEnvironment(command string) string {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return ""
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ""
	}
	first := filepath.Base(fields[0])
	// go run: first token is "go", second is "run"
	if first == "go" && len(fields) >= 2 && fields[1] == "run" {
		return "go"
	}
	switch first {
	case "npm":
		return "npm"
	case "yarn":
		return "yarn"
	case "pnpm":
		return "pnpm"
	case "poetry":
		return "poetry"
	case "pipenv":
		return "pipenv"
	case "cargo":
		return "cargo"
	}
	return ""
}
