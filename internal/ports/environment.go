package ports

import (
	"path/filepath"
	"strings"
)

// DetectEnvironment returns how the process was launched or which runtime it is
// (npm, yarn, pnpm, poetry, pipenv, cargo, go, or node, python, ruby, java, dotnet).
// Empty string if not detected. Uses first token of command (supports paths like /usr/bin/npm).
// When the listening process is the runtime (e.g. node), we show the runtime so the ENV column is useful.
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

	// Launchers (first token)
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
	case "bun":
		return "bun"
	case "deno":
		return "deno"
	case "uv":
		return "uv"
	}

	// Runtimes: process is often the runtime (node, python, etc.), not the launcher
	switch first {
	case "node", "nodejs":
		return "node"
	case "python", "python3", "python2":
		return "python"
	case "ruby", "ruby30", "ruby31":
		return "ruby"
	case "java":
		return "java"
	case "dotnet":
		return "dotnet"
	}
	// Infer from command when binary has a version suffix (e.g. python3.11)
	if strings.HasPrefix(first, "python") {
		return "python"
	}
	if strings.HasPrefix(first, "ruby") {
		return "ruby"
	}
	if strings.HasPrefix(first, "node") {
		return "node"
	}
	return ""
}
