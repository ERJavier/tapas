package ports

import "strings"

// System port range: 1-1023 (privileged ports on Unix; require root to bind).
// Well-known system services typically use this range (e.g. 22 SSH, 53 DNS, 80 HTTP, 443 HTTPS).

// IsSystemPort returns true if the port is in the system (privileged) range.
// Used for UI indicators and row styling so system ports are clearly distinguished.
func IsSystemPort(port uint16) bool {
	return port > 0 && port < 1024
}

// IsSystemProcess returns true if the process is a known OS-level system daemon
// (e.g. macOS: rapportd, sharingd, identitys, Control Center). Used together with
// IsSystemPort so system ports and system processes both get the system indicator.
func IsSystemProcess(process string) bool {
	proc := strings.ToLower(strings.TrimSpace(process))
	if proc == "" {
		return false
	}
	// macOS system daemons (AirDrop, Handoff, iCloud, Control Center, Remote, Replication)
	if strings.Contains(proc, "sharingd") {
		return true
	}
	if strings.Contains(proc, "rapportd") {
		return true
	}
	if strings.Contains(proc, "identitys") || strings.Contains(proc, "identityservicesd") {
		return true
	}
	if strings.Contains(proc, "controlcenter") || strings.Contains(proc, "controlce") {
		return true
	}
	if strings.Contains(proc, "remotepairingd") || strings.Contains(proc, "remotepai") {
		return true
	}
	if strings.Contains(proc, "replicat") {
		return true
	}
	return false
}
