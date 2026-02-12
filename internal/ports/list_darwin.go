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
		port, ok := portFromLsofName(name)
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
		list = append(list, Port{
			PortNum:    uint16(port),
			PID:        pid,
			Process:    process,
			Protocol:   "tcp",
			StartTime:  startTime,
			WorkingDir: workingDir,
			Command:    command,
			Framework:  DetectFramework(workingDir, command, process),
			InDocker:   isDocker(pid),
		})
	}
	return list, sc.Err()
}

func portFromLsofName(name string) (int, bool) {
	// *:3000 (LISTEN) or *:7000 (no paren when from single field)
	i := strings.Index(name, ":")
	if i < 0 {
		return 0, false
	}
	rest := name[i+1:]
	j := strings.Index(rest, " ")
	if j < 0 {
		j = strings.Index(rest, ")")
	}
	var s string
	if j >= 0 {
		s = rest[:j]
	} else {
		s = rest
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	port, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return port, true
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
