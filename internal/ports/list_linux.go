//go:build linux

package ports

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func init() {
	defaultLister = &linuxLister{}
}

type linuxLister struct{}

func (l *linuxLister) List() ([]Port, error) {
	cmd := exec.Command("ss", "-tlnp")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	list, err := parseSS(out)
	if err != nil {
		return nil, err
	}
	EnrichDocker(&list)
	EnrichConnectionCounts(&list)
	return list, nil
}

// parseSS parses ss -tlnp. Format:
// State  Recv-Q Send-Q Local Address:Port Peer Address:Port Process
// LISTEN 0      128    *:3000             *:*    users:(("node",pid=123,fd=20))
func parseSS(out []byte) ([]Port, error) {
	var list []Port
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Scan() // header
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		if fields[0] != "LISTEN" {
			continue
		}
		addrStr := fields[3]
		port, ok := portFromSSAddr(addrStr)
		if !ok {
			continue
		}
		pid, process := pidAndProcessFromSS(line)
		startTime, _ := processStartTimeLinux(pid)
		workingDir := getWorkingDirLinux(pid)
		command := getCommandLinux(pid)
		bindAddr := bindFromSSAddr(addrStr)
		if bindAddr == "*" {
			bindAddr = "0.0.0.0"
		}
		list = append(list, Port{
			PortNum:           uint16(port),
			PID:               pid,
			Process:           process,
			Protocol:          "tcp",
			StartTime:         startTime,
			WorkingDir:        workingDir,
			Command:           command,
			Framework:         DetectFramework(workingDir, command, process),
			InDocker:          isDocker(pid),
			BindAddress:       bindAddr,
			ProjectDisplayName: ProjectDisplayName(workingDir),
		})
	}
	return list, sc.Err()
}

func portFromSSAddr(addr string) (int, bool) {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return 0, false
	}
	port, err := strconv.Atoi(addr[i+1:])
	if err != nil {
		return 0, false
	}
	return port, true
}

func bindFromSSAddr(addr string) string {
	i := strings.LastIndex(addr, ":")
	if i < 0 {
		return ""
	}
	return strings.TrimSpace(addr[:i])
}

func pidAndProcessFromSS(line string) (int, string) {
	// users:(("node",pid=123,fd=20)) or empty
	i := strings.Index(line, "pid=")
	if i < 0 {
		return 0, "—"
	}
	i += 4
	j := i
	for j < len(line) && line[j] >= '0' && line[j] <= '9' {
		j++
	}
	pid, _ := strconv.Atoi(line[i:j])
	process := "—"
	comm := strings.Index(line, "\"")
	if comm >= 0 {
		end := strings.Index(line[comm+1:], "\"")
		if end >= 0 {
			process = line[comm+1 : comm+1+end]
		}
	}
	return pid, process
}

func processStartTimeLinux(pid int) (time.Time, error) {
	if pid <= 0 {
		return time.Time{}, nil
	}
	data, err := readFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return time.Time{}, err
	}
	// Field 22 is starttime (jiffies since boot)
	start := 0
	for n := 0; n < 21; n++ {
		i := strings.Index(data[start:], " ")
		if i < 0 {
			return time.Time{}, nil
		}
		start += i + 1
	}
	end := start
	for end < len(data) && data[end] != ' ' {
		end++
	}
	jiffies, err := strconv.ParseUint(data[start:end], 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	// Elapsed since process start: system uptime - process start time in seconds
	uptimeSec, err := readUptimeSeconds()
	if err != nil {
		return time.Time{}, err
	}
	clkTck := 100.0
	startSec := float64(jiffies) / clkTck
	elapsed := uptimeSec - startSec
	return time.Now().Add(-time.Duration(elapsed * float64(time.Second))), nil
}

func readUptimeSeconds() (float64, error) {
	data, err := readFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	i := 0
	for i < len(data) && data[i] != ' ' {
		i++
	}
	return strconv.ParseFloat(strings.TrimSpace(data[:i]), 64)
}

func getWorkingDirLinux(pid int) string {
	if pid <= 0 {
		return ""
	}
	path, err := os.Readlink("/proc/" + strconv.Itoa(pid) + "/cwd")
	if err != nil {
		return ""
	}
	return path
}

func getCommandLinux(pid int) string {
	if pid <= 0 {
		return ""
	}
	cmdline, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/cmdline")
	if err != nil {
		return ""
	}
	return strings.ReplaceAll(strings.TrimSpace(string(cmdline)), "\x00", " ")
}

func readFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
