package ports

import (
	"fmt"
	"os"
	"syscall"
)

// KillResult is the result of a kill attempt.
type KillResult struct {
	OK    bool
	Error string
}

// Kill sends SIGTERM to the process identified by pid.
// Caller is responsible for confirmation; this package does not prompt.
func Kill(pid int) KillResult {
	if pid <= 0 {
		return KillResult{OK: false, Error: "invalid pid"}
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return KillResult{OK: false, Error: err.Error()}
	}
	err = proc.Signal(syscall.SIGTERM)
	if err != nil {
		return KillResult{OK: false, Error: err.Error()}
	}
	return KillResult{OK: true}
}

// KillPort kills the process listening on the given port by PID.
// port is only used for error messages; the actual target is pid.
func KillPort(port uint16, pid int) KillResult {
	r := Kill(pid)
	if !r.OK && r.Error != "" {
		r.Error = fmt.Sprintf("Failed to kill port %d (%s)", port, r.Error)
	}
	return r
}
