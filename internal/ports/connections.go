package ports

// EnrichConnectionCounts sets ConnectionCount on each port from established connection counts.
// getConnectionCounts is implemented per-OS (darwin/linux); no-op if unavailable.
func EnrichConnectionCounts(ports *[]Port) {
	if ports == nil || len(*ports) == 0 {
		return
	}
	counts := getConnectionCounts()
	if len(counts) == 0 {
		return
	}
	for i := range *ports {
		(*ports)[i].ConnectionCount = counts[(*ports)[i].PortNum]
	}
}
