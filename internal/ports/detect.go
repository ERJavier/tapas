package ports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// DetectFramework returns a framework label from working dir, full command, and process name.
// Returns "" if unknown. Order: Next.js, Vite, Rails, Django, Node (specific over generic).
func DetectFramework(workingDir, command, process string) string {
	cmd := strings.ToLower(command)
	proc := strings.ToLower(process)
	if workingDir != "" {
		if f := detectFromPackageJSON(workingDir); f != "" {
			return f
		}
		if f := detectFromRails(workingDir); f != "" {
			return f
		}
		if f := detectFromDjango(workingDir); f != "" {
			return f
		}
	}
	// Command/process heuristics when no project dir or no file-based match
	if strings.Contains(cmd, "next") || proc == "next" {
		return "Next.js"
	}
	if strings.Contains(cmd, "vite") || proc == "vite" {
		return "Vite"
	}
	if strings.Contains(cmd, "rails") || strings.Contains(cmd, "bin/rails") || proc == "rails" {
		return "Rails"
	}
	if strings.Contains(cmd, "manage.py") || strings.Contains(cmd, "django") || (proc == "python" && strings.Contains(cmd, "manage")) {
		return "Django"
	}
	if proc == "node" || strings.Contains(cmd, " node ") || strings.HasPrefix(cmd, "node ") {
		return "Node"
	}
	return ""
}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
}

func detectFromPackageJSON(workingDir string) string {
	data, err := os.ReadFile(filepath.Join(workingDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg packageJSON
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
	if _, ok := allDeps["react"]; ok {
		return "Node"
	}
	// Any package.json project without a more specific framework
	return "Node"
}

func detectFromRails(workingDir string) string {
	_, err := os.Stat(filepath.Join(workingDir, "Gemfile"))
	if err != nil {
		return ""
	}
	gemfile, err := os.ReadFile(filepath.Join(workingDir, "Gemfile"))
	if err != nil {
		return ""
	}
	if strings.Contains(strings.ToLower(string(gemfile)), "rails") {
		return "Rails"
	}
	return ""
}

func detectFromDjango(workingDir string) string {
	if _, err := os.Stat(filepath.Join(workingDir, "manage.py")); err == nil {
		return "Django"
	}
	return ""
}
