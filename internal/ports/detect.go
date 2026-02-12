package ports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectFramework returns a framework label: high value, low noise.
// Prefer full command (what's actually running), then working dir files.
// Returns "" when unknown. Order: specific over generic (Next.js > Node, etc.).
func DetectFramework(workingDir, command, process string) string {
	cmd := strings.ToLower(command)
	proc := strings.ToLower(strings.TrimSpace(process))

	// 1. Command-first: what's actually running (high signal)
	if f := fromCommand(cmd, proc); f != "" {
		return f
	}

	// 2. Working dir: project files (supporting signal)
	if workingDir != "" {
		if f := fromPackageJSON(workingDir); f != "" {
			return f
		}
		if f := fromRails(workingDir); f != "" {
			return f
		}
		if f := fromDjango(workingDir); f != "" {
			return f
		}
		if f := fromGoMod(workingDir); f != "" {
			return f
		}
	}

	return ""
}

// fromCommand detects framework from full command (ps) and process name.
// Look for: next dev, vite, rails server, uvicorn, gunicorn, npm run dev, yarn dev.
func fromCommand(cmd, proc string) string {
	// Next.js: next dev, next start
	if strings.Contains(cmd, "next dev") || strings.Contains(cmd, "next start") || proc == "next" {
		return "Next.js"
	}
	// Vite
	if strings.Contains(cmd, "vite") || proc == "vite" {
		return "Vite"
	}
	// Rails: rails server, rails s, bin/rails
	if strings.Contains(cmd, "rails server") || strings.Contains(cmd, "rails s ") ||
		strings.Contains(cmd, "bin/rails") || proc == "rails" {
		return "Rails"
	}
	// Django / Python ASGI-WSGI: uvicorn, gunicorn
	if strings.Contains(cmd, "uvicorn") || strings.Contains(cmd, "gunicorn") {
		return "Django"
	}
	// manage.py runserver
	if strings.Contains(cmd, "manage.py") && strings.Contains(cmd, "runserver") {
		return "Django"
	}
	// Generic Node dev: npm run dev, yarn dev, pnpm dev
	if strings.Contains(cmd, "npm run dev") || strings.Contains(cmd, "yarn dev") ||
		strings.Contains(cmd, "pnpm dev") || strings.Contains(cmd, "pnpm run dev") {
		return "Node"
	}
	// Fallback: node process without a more specific signal (low noise: only if command suggests dev server)
	if (proc == "node" || strings.HasPrefix(cmd, "node ")) && (strings.Contains(cmd, "dev") || strings.Contains(cmd, "start")) {
		return "Node"
	}
	return ""
}

func fromPackageJSON(workingDir string) string {
	data, err := os.ReadFile(filepath.Join(workingDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}
	if _, ok := allDeps["next"]; ok {
		return "Next.js"
	}
	if _, ok := allDeps["vite"]; ok {
		return "Vite"
	}
	for _, script := range pkg.Scripts {
		s := strings.ToLower(script)
		if strings.Contains(s, "next") && (strings.Contains(s, "dev") || strings.Contains(s, "start")) {
			return "Next.js"
		}
		if strings.Contains(s, "vite") {
			return "Vite"
		}
	}
	// Only label Node from package.json if we have a clear dev script (low noise)
	for _, script := range pkg.Scripts {
		s := strings.ToLower(script)
		if strings.Contains(s, "dev") || strings.Contains(s, "start") {
			return "Node"
		}
	}
	return ""
}

func fromRails(workingDir string) string {
	_, err := os.Stat(filepath.Join(workingDir, "Gemfile"))
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(workingDir, "Gemfile"))
	if err != nil {
		return ""
	}
	if strings.Contains(strings.ToLower(string(data)), "rails") {
		return "Rails"
	}
	return ""
}

func fromDjango(workingDir string) string {
	if _, err := os.Stat(filepath.Join(workingDir, "manage.py")); err == nil {
		return "Django"
	}
	// requirements.txt with django or uvicorn/gunicorn
	if _, err := os.Stat(filepath.Join(workingDir, "requirements.txt")); err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(workingDir, "requirements.txt"))
	if err != nil {
		return ""
	}
	s := strings.ToLower(string(data))
	if strings.Contains(s, "django") || strings.Contains(s, "uvicorn") || strings.Contains(s, "gunicorn") {
		return "Django"
	}
	return ""
}

func fromGoMod(workingDir string) string {
	if _, err := os.Stat(filepath.Join(workingDir, "go.mod")); err != nil {
		return ""
	}
	return "Go"
}
