//go:build !darwin && !linux

package ports

// getConnectionCounts returns nil on unsupported platforms (no connection count available).
func getConnectionCounts() map[uint16]int {
	return nil
}
