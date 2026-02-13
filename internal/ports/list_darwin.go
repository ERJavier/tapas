//go:build darwin

package ports

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func init() {
	defaultLister = &darwinLister{}
}

type darwinLister struct{}

func (d *darwinLister) List() ([]Port, error) {
	cmd := exec.Command("lsof", "-i", "-P", "-n", "-sTCP:LISTEN")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	list, err := parseLsof(out)
	if err != nil {
		return nil, err
	}
	EnrichDocker(&list)
	EnrichConnectionCounts(&list)
	return list, nil
}

// parseLsof parses lsof -i -P -n output. Columns: COMMAND, PID, USER, FD, TYPE, DEVICE, SIZE/OFF, NODE, NAME
// NAME is like *:3000 (LISTEN). We need PORT from NAME and PID.
func parseLsof(out []byte) ([]Port, error) {
	var list []Port
	seen := make(map[string]bool) // "pid:port" to avoid dupes from multiple FDs
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Scan() // skip header
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		name := fields[8]
		addr, port, ok := addrPortFromLsofName(name)
		if !ok {
			continue
		}
		key := strconv.Itoa(pid) + ":" + strconv.Itoa(port)
		if seen[key] {
			continue
		}
		seen[key] = true
		startTime, _ := processStartTime(pid)
		workingDir := getWorkingDir(pid)
		command := getCommand(pid)
		process := fields[0]
		bindAddr := addr
		if bindAddr == "*" {
			bindAddr = "0.0.0.0"
		}
		list = append(list, Port{
			PortNum:            uint16(port),
			PID:                pid,
			Process:            process,
			Protocol:           "tcp",
			StartTime:          startTime,
			WorkingDir:         workingDir,
			Command:            command,
			Framework:          DetectFramework(workingDir, command, process),
			InDocker:           isDocker(pid),
			BindAddress:        bindAddr,
			ProjectDisplayName: ProjectDisplayName(workingDir),
			Environment:        DetectEnvironment(command),
		})
	}
	return list, sc.Err()
}

// addrPortFromLsofName parses NAME like "*:3000 (LISTEN)" or "127.0.0.1:3000 (LISTEN)". Returns (address, port, ok).
func addrPortFromLsofName(name string) (addr string, port int, ok bool) {
	i := strings.Index(name, ":")
	if i < 0 {
		return "", 0, false
	}
	addr = strings.TrimSpace(name[:i])
	rest := name[i+1:]
	j := strings.Index(rest, " ")
	if j < 0 {
		j = strings.Index(rest, ")")
	}
	var portStr string
	if j >= 0 {
		portStr = strings.TrimSpace(rest[:j])
	} else {
		portStr = strings.TrimSpace(rest)
	}
	if portStr == "" {
		return "", 0, false
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, false
	}
	return addr, port, true
}

func processStartTime(pid int) (time.Time, error) {
	cmd := exec.Command("ps", "-o", "lstart=", "-p", strconv.Itoa(pid))
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}, err
	}
	s := strings.TrimSpace(string(out))
	if s == "" {
		return time.Time{}, nil
	}
	// "Mon Jan  2 15:04:05 2006"
	t, err := time.Parse("Mon Jan 2 15:04:05 2006", s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func getWorkingDir(pid int) string {
	cmd := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	s := string(out)
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(line, "n") {
			dir := strings.TrimPrefix(line, "n")
			return strings.TrimSpace(dir)
		}
	}
	return ""
}

func getCommand(pid int) string {
	cmd := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(pid))
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
