package usecase

import (
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cbout22/copilot-sync/internal/config"
)

// mockFS implements port.FileSystem using in-memory storage.
type mockFS struct {
	mu    sync.Mutex
	files map[string][]byte
	dirs  map[string]bool
}

func newMockFS() *mockFS {
	return &mockFS{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

func (m *mockFS) ReadFile(path string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, ok := m.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return append([]byte(nil), data...), nil
}

func (m *mockFS) WriteFile(path string, data []byte, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = append([]byte(nil), data...)
	return nil
}

func (m *mockFS) MkdirAll(path string, _ fs.FileMode) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.dirs[path] = true
	return nil
}

func (m *mockFS) Remove(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	return nil
}

func (m *mockFS) RemoveAll(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k := range m.files {
		if k == path || strings.HasPrefix(k, path+"/") {
			delete(m.files, k)
		}
	}
	delete(m.dirs, path)
	return nil
}

func (m *mockFS) Stat(path string) (fs.FileInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.files[path]; ok {
		return mockFileInfo{name: path, isDir: false}, nil
	}
	if _, ok := m.dirs[path]; ok {
		return mockFileInfo{name: path, isDir: true}, nil
	}
	return nil, os.ErrNotExist
}

type mockFileInfo struct {
	name  string
	isDir bool
}

func (m mockFileInfo) Name() string      { return m.name }
func (m mockFileInfo) Size() int64       { return 0 }
func (m mockFileInfo) Mode() fs.FileMode { return 0644 }
func (m mockFileInfo) ModTime() time.Time { return time.Time{} }
func (m mockFileInfo) IsDir() bool       { return m.isDir }
func (m mockFileInfo) Sys() any          { return nil }

// mockGitHub implements port.GitHubResolver for testing.
type mockGitHub struct {
	files map[string][]byte // key: "org/repo/path@ref"
	sha   string
}

func (m *mockGitHub) ResolveRef(ref config.AssetRef) (config.AssetRef, error) {
	return ref, nil
}

func (m *mockGitHub) DownloadFile(ref config.AssetRef) ([]byte, error) {
	key := ref.Raw()
	if content, ok := m.files[key]; ok {
		return content, nil
	}
	return nil, fmt.Errorf("file not found: %s", key)
}

func (m *mockGitHub) ListDirectory(ref config.AssetRef) ([]config.GitHubTreeEntry, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockGitHub) ResolveSHA(ref config.AssetRef) (string, error) {
	return m.sha, nil
}
