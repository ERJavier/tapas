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
	return parseLsof(out)
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
		list = append(list, Port{
			PortNum:   uint16(port),
			PID:       pid,
			Process:   fields[0],
			Protocol:  "tcp",
			StartTime: startTime,
			WorkingDir: getWorkingDir(pid),
			Command:   getCommand(pid),
		})
	}
	return list, sc.Err()
}

func portFromLsofName(name string) (int, bool) {
	// *:3000 (LISTEN) or [::]:3000 (LISTEN)
	i := strings.Index(name, ":")
	if i < 0 {
		return 0, false
	}
	j := strings.Index(name[i+1:], " ")
	if j < 0 {
		j = strings.Index(name[i+1:], ")")
	}
	if j < 0 {
		return 0, false
	}
	s := name[i+1 : i+1+j]
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
