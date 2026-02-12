//go:build darwin

package ports

// isDocker on macOS: containers run in a VM so host PIDs are not in a cgroup.
// We could match by process name (e.g. com.docker.backend) but that's the daemon, not the app.
// Return false for now; Docker detection is best-effort on Linux.
func isDocker(pid int) bool {
	return false
}
