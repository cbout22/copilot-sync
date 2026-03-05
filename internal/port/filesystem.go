package port

import (
	"io/fs"
	"os"
)

// FileSystem abstracts file I/O operations.
type FileSystem interface {
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Stat(path string) (fs.FileInfo, error)
}

// OSFileSystem is the production adapter backed by the os package.
type OSFileSystem struct{}

func (OSFileSystem) ReadFile(path string) ([]byte, error)                       { return os.ReadFile(path) }
func (OSFileSystem) WriteFile(path string, data []byte, perm fs.FileMode) error { return os.WriteFile(path, data, perm) }
func (OSFileSystem) MkdirAll(path string, perm fs.FileMode) error              { return os.MkdirAll(path, perm) }
func (OSFileSystem) Remove(path string) error                                   { return os.Remove(path) }
func (OSFileSystem) RemoveAll(path string) error                                { return os.RemoveAll(path) }
func (OSFileSystem) Stat(path string) (fs.FileInfo, error)                      { return os.Stat(path) }
