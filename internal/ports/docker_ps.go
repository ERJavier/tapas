package ports

import (
	"os/exec"
	"strconv"
	"strings"
)

// dockerPortMap returns host port -> {container name, image} from docker ps.
// Ignores errors (docker not installed or not running); returns nil map on failure.
func dockerPortMap() map[uint16]struct{ Name, Image string } {
	cmd := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Image}}\t{{.Ports}}")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	m := make(map[uint16]struct{ Name, Image string })
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		image := strings.TrimSpace(parts[1])
		portsStr := parts[2]
		for _, segment := range strings.Split(portsStr, ",") {
			segment = strings.TrimSpace(segment)
			hostPort := parseDockerHostPort(segment)
			if hostPort > 0 {
				m[hostPort] = struct{ Name, Image string }{Name: name, Image: image}
			}
		}
	}
	return m
}

// parseDockerHostPort extracts host port from a segment like "0.0.0.0:3000->3000/tcp" or ":::5432->5432/tcp".
// Returns 0 if not parseable.
func parseDockerHostPort(segment string) uint16 {
	i := strings.Index(segment, "->")
	if i < 0 {
		return 0
	}
	left := strings.TrimSpace(segment[:i])
	if left == "" {
		return 0
	}
	// Host part is "0.0.0.0:3000" or ":::3000" - port is after last ':'
	lastColon := strings.LastIndex(left, ":")
	if lastColon < 0 {
		return 0
	}
	portStr := strings.TrimSpace(left[lastColon+1:])
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		return 0
	}
	return uint16(port)
}

// EnrichDocker fills DockerContainerName, DockerImage, and InDocker for ports that match docker ps mappings.
// Call after building the port list. No-op if docker is unavailable.
func EnrichDocker(ports *[]Port) {
	if ports == nil || len(*ports) == 0 {
		return
	}
	m := dockerPortMap()
	if len(m) == 0 {
		return
	}
	for i := range *ports {
		p := &(*ports)[i]
		if info, ok := m[p.PortNum]; ok {
			p.DockerContainerName = info.Name
			p.DockerImage = info.Image
			p.InDocker = true
		}
	}
}
