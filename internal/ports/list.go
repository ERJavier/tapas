package ports

// Lister lists listening ports. Implementations are OS-specific.
type Lister interface {
	List() ([]Port, error)
}

var defaultLister Lister

// DefaultLister returns the appropriate lister for the current OS.
// Callers in internal/ui must use this (or a passed Lister); UI must not run OS commands directly.
func DefaultLister() Lister {
	return defaultLister
}
