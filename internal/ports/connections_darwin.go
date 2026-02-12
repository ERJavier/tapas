//go:build darwin

package ports

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// getConnectionCounts returns established TCP connection count per local port (darwin: netstat).
func getConnectionCounts() map[uint16]int {
	cmd := exec.Command("netstat", "-an", "-p", "tcp")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	counts := make(map[uint16]int)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := sc.Text()
		if !strings.Contains(line, "ESTABLISHED") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local address is typically 4th field: 127.0.0.1.3000 or *.3000 (BSD style: IP.port)
		local := fields[3]
		port := portFromNetstatLocal(local)
		if port > 0 {
			counts[port]++
		}
	}
	return counts
}

func portFromNetstatLocal(local string) uint16 {
	// 127.0.0.1.3000 or *.3000
	i := strings.LastIndex(local, ".")
	if i < 0 {
		return 0
	}
	p, err := strconv.Atoi(local[i+1:])
	if err != nil || p <= 0 || p > 65535 {
		return 0
	}
	return uint16(p)
}
