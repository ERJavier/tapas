package ports

import "time"

// Port holds metadata for a listening port and its process.
type Port struct {
	PortNum    uint16
	PID        int
	Process    string
	Protocol   string
	StartTime  time.Time
	WorkingDir string
	Command    string

	// v1.0 Smart Detection
	Framework string // e.g. "Next.js", "Vite", "Rails", "Django", "Node", ""
	InDocker  bool   // process is running inside a Docker/containerd container
}

// Uptime returns the duration since StartTime. If StartTime is zero, returns 0.
func (p *Port) Uptime() time.Duration {
	if p.StartTime.IsZero() {
		return 0
	}
	return time.Since(p.StartTime)
}

// Project returns a short label for the project (working directory).
// Returns the last element of the path or "—" if empty.
func (p *Port) Project() string {
	if p.WorkingDir == "" {
		return "—"
	}
	// Use last path component
	for i := len(p.WorkingDir) - 1; i >= 0; i-- {
		if p.WorkingDir[i] == '/' {
			if i+1 < len(p.WorkingDir) {
				return p.WorkingDir[i+1:]
			}
			return p.WorkingDir
		}
	}
	return p.WorkingDir
}
