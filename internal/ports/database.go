package ports

// DatabaseProductName returns the product name for common DB ports, or "" if not a known DB port.
func DatabaseProductName(port uint16) string {
	switch port {
	case 5432:
		return "PostgreSQL"
	case 3306:
		return "MySQL"
	case 6379:
		return "Redis"
	case 27017:
		return "Mongo"
	case 9200:
		return "Elasticsearch"
	default:
		return ""
	}
}
