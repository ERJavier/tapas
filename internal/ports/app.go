package ports

import "strings"

// AppName returns a short label for common apps by port and/or process name.
// Used when Framework and Docker are not set. Prefer port (more reliable), then process.
// Returns "" when unknown.
func AppName(port uint16, process string) string {
	proc := strings.ToLower(strings.TrimSpace(process))

	// 1. Port-based: well-known services
	if name := appByPort(port); name != "" {
		return name
	}

	// 2. Process-name-based: common binaries and names
	return appByProcess(proc)
}

// appByPort returns app name for well-known ports.
func appByPort(port uint16) string {
	switch port {
	// Databases
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
	case 5984:
		return "CouchDB"
	case 11211:
		return "Memcached"
	case 9042:
		return "Cassandra"
	case 27018:
		return "Mongo"
	case 1433:
		return "SQL Server"
	// Message queues / brokers
	case 5672:
		return "RabbitMQ"
	case 15672:
		return "RabbitMQ"
	case 61613, 61614:
		return "ActiveMQ"
	case 9092:
		return "Kafka"
	// Web / dev servers (only when no framework matched)
	case 80, 443, 8080, 8443, 8000, 8888:
		return "" // leave to process: nginx, Apache, etc.
	// Monitoring / dashboards
	case 5601:
		return "Kibana"
	case 3000, 3001, 5000, 5001, 5173, 5174:
		return "" // dev ports; framework or process will usually identify
	case 1313:
		return "Hugo"
	case 2368:
		return "Ghost"
	// Mail
	case 25, 587, 465:
		return "SMTP"
	case 993:
		return "IMAP"
	case 995:
		return "POP3"
	// Other common services
	case 389, 636:
		return "LDAP"
	case 53:
		return "DNS"
	case 22:
		return "SSH"
	default:
		return ""
	}
}

// appByProcess returns app name from process (binary) name.
func appByProcess(proc string) string {
	if proc == "" {
		return ""
	}
	// Databases and stores
	if strings.Contains(proc, "postgres") || strings.HasPrefix(proc, "pg_") {
		return "PostgreSQL"
	}
	if strings.Contains(proc, "redis") {
		return "Redis"
	}
	if strings.Contains(proc, "mongo") {
		return "Mongo"
	}
	if strings.Contains(proc, "mysql") || strings.Contains(proc, "mariadb") {
		return "MySQL"
	}
	if strings.Contains(proc, "elastic") {
		return "Elasticsearch"
	}
	if strings.Contains(proc, "memcached") {
		return "Memcached"
	}
	if strings.Contains(proc, "couchdb") {
		return "CouchDB"
	}
	// Web servers and proxies
	if strings.Contains(proc, "nginx") {
		return "nginx"
	}
	if strings.Contains(proc, "apache") || strings.Contains(proc, "httpd") {
		return "Apache"
	}
	if strings.Contains(proc, "caddy") {
		return "Caddy"
	}
	// Runtimes (only when no framework matched)
	if proc == "node" || strings.HasPrefix(proc, "node ") {
		return "Node"
	}
	if strings.Contains(proc, "ruby") || proc == "rails" {
		return "Ruby"
	}
	if strings.Contains(proc, "python") || strings.Contains(proc, "uvicorn") || strings.Contains(proc, "gunicorn") {
		return "Python"
	}
	if strings.Contains(proc, "java") || strings.Contains(proc, "gradle") {
		return "Java"
	}
	if strings.Contains(proc, "dotnet") {
		return ".NET"
	}
	// Message queues
	if strings.Contains(proc, "rabbitmq") {
		return "RabbitMQ"
	}
	if strings.Contains(proc, "kafka") {
		return "Kafka"
	}
	// Common macOS / system (keep label short and recognizable)
	if strings.Contains(proc, "sharingd") {
		return "AirDrop"
	}
	if strings.Contains(proc, "rapportd") {
		return "Handoff"
	}
	if strings.Contains(proc, "identity") || strings.Contains(proc, "identitys") {
		return "iCloud"
	}
	if strings.Contains(proc, "replicat") {
		return "Replication"
	}
	if strings.Contains(proc, "controlcenter") || strings.Contains(proc, "controlce") {
		return "Control Center"
	}
	if strings.Contains(proc, "creative") && strings.Contains(proc, "cloud") {
		return "Creative Cloud"
	}
	if strings.Contains(proc, "adobe") {
		return "Adobe"
	}
	if strings.Contains(proc, "cursor") {
		return "Cursor"
	}
	if strings.Contains(proc, "code") && (strings.Contains(proc, "visual") || strings.Contains(proc, "vs")) {
		return "VS Code"
	}
	if strings.Contains(proc, "ollama") {
		return "Ollama"
	}
	// Remote / dev tools
	if strings.Contains(proc, "remotepai") {
		return "Remote"
	}
	return ""
}
