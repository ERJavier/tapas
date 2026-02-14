package ports

import (
	"fmt"
	"strings"
	"syscall"
)

// KillResult is the result of a kill attempt.
type KillResult struct {
	OK    bool
	Error string
}

// Kill sends SIGTERM to the process identified by pid.
// Uses syscall.Kill directly so it works reliably on macOS (os.FindProcess+Signal can fail there).
// Caller is responsible for confirmation; this package does not prompt.
func Kill(pid int) KillResult {
	if pid <= 0 {
		return KillResult{OK: false, Error: "invalid pid"}
	}
	err := syscall.Kill(pid, syscall.SIGTERM)
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "permission") || strings.Contains(strings.ToLower(msg), "operation not permitted") {
			msg = "permission denied (try running TAPAS with sudo to kill system processes)"
		}
		if strings.Contains(strings.ToLower(msg), "no such process") || strings.Contains(strings.ToLower(msg), "esrch") {
			msg = "process already exited"
		}
		return KillResult{OK: false, Error: msg}
	}
	return KillResult{OK: true}
}

// KillPort kills the process listening on the given port by PID (sends SIGTERM).
// port is only used for error messages; the actual target is pid.
func KillPort(port uint16, pid int) KillResult {
	r := Kill(pid)
	if !r.OK && r.Error != "" {
		r.Error = fmt.Sprintf("Failed to kill port %d (%s)", port, r.Error)
	}
	return r
}

// KillPortForce kills the process with SIGKILL (cannot be ignored by the process).
// Use when SIGTERM does nothing (e.g. some GUI apps like Adobe).
func KillPortForce(port uint16, pid int) KillResult {
	if pid <= 0 {
		return KillResult{OK: false, Error: fmt.Sprintf("Failed to kill port %d (invalid pid)", port)}
	}
	err := syscall.Kill(pid, syscall.SIGKILL)
	if err != nil {
		msg := err.Error()
		if strings.Contains(strings.ToLower(msg), "permission") || strings.Contains(strings.ToLower(msg), "operation not permitted") {
			msg = "permission denied (try running TAPAS with sudo)"
		}
		if strings.Contains(strings.ToLower(msg), "no such process") || strings.Contains(strings.ToLower(msg), "esrch") {
			msg = "process already exited"
		}
		return KillResult{OK: false, Error: fmt.Sprintf("Failed to kill port %d (%s)", port, msg)}
	}
	return KillResult{OK: true}
}
