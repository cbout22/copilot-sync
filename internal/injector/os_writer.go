package injector

import "os"

// OSFileWriter implements FileWriter using the real filesystem.
type OSFileWriter struct{}

var _ FileWriter = (*OSFileWriter)(nil)

func (w *OSFileWriter) Write(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

func (w *OSFileWriter) MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func (w *OSFileWriter) Remove(path string) error {
	return os.RemoveAll(path)
}

func (w *OSFileWriter) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
