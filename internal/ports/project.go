package ports

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// ProjectDisplayName returns a short, TAPAS-style project label from the working directory.
// Prefer package.json "name", then .git repo (directory name), then docker-compose.yml (directory name), then empty.
func ProjectDisplayName(workingDir string) string {
	if workingDir == "" {
		return ""
	}
	if name := nameFromPackageJSON(workingDir); name != "" {
		return name
	}
	if name := nameFromGit(workingDir); name != "" {
		return name
	}
	if name := nameFromDockerCompose(workingDir); name != "" {
		return name
	}
	return ""
}

func nameFromPackageJSON(workingDir string) string {
	data, err := os.ReadFile(filepath.Join(workingDir, "package.json"))
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return ""
	}
	return strings.TrimSpace(pkg.Name)
}

func nameFromGit(workingDir string) string {
	info, err := os.Stat(filepath.Join(workingDir, ".git"))
	if err != nil {
		return ""
	}
	if !info.IsDir() {
		return ""
	}
	return lastPathComponent(workingDir)
}

func nameFromDockerCompose(workingDir string) string {
	_, err := os.Stat(filepath.Join(workingDir, "docker-compose.yml"))
	if err != nil {
		return ""
	}
	return lastPathComponent(workingDir)
}

func lastPathComponent(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i+1 < len(path) {
				return path[i+1:]
			}
			return path
		}
	}
	return path
}
