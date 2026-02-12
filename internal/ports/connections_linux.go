//go:build linux

package ports

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
)

// getConnectionCounts returns established TCP connection count per local port (linux: ss).
func getConnectionCounts() map[uint16]int {
	cmd := exec.Command("ss", "-tn", "state", "established")
	cmd.Env = []string{"LC_ALL=C"}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	counts := make(map[uint16]int)
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	sc.Scan() // skip header
	for sc.Scan() {
		line := sc.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local Address:Port is typically 4th field (e.g. 127.0.0.1:3000)
		local := fields[3]
		port, ok := portFromSSAddr(local)
		if ok && port > 0 {
			counts[uint16(port)]++
		}
	}
	return counts
}
