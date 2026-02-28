package injector

// FileWriter abstracts filesystem operations for asset injection.
type FileWriter interface {
	// Write creates or overwrites a file at the given path with the given data.
	Write(path string, data []byte) error

	// MkdirAll creates a directory path and all necessary parents.
	MkdirAll(path string) error

	// Remove deletes a file or directory (recursively).
	Remove(path string) error

	// Exists reports whether the given path exists.
	Exists(path string) bool
}
