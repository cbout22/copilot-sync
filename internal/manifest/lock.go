package manifest

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const DefaultLockFile = ".cops.lock"

// LockFile is the shadow manifest that tracks which files cops "owns".
// It stores the resolved state of each asset so that `cops sync` and
// `cops check` can detect drift.
type LockFile struct {
	// Version of the lock file format.
	Version int `json:"version"`
	// Entries keyed by "<type>/<name>".
	Entries map[string]LockEntry `json:"entries"`
}

// LockEntry records the resolved state of a single managed asset.
type LockEntry struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Ref         string `json:"ref"`          // original ref string (e.g. org/repo/path@v1.2)
	ResolvedSHA string `json:"resolved_sha"` // commit SHA the ref resolved to at sync time
	TargetPath  string `json:"target_path"`  // local file/dir path relative to project root
	Checksum    string `json:"checksum"`     // SHA-256 of the downloaded content
	SyncedAt    string `json:"synced_at"`    // RFC 3339 timestamp of last sync
}

// NewLockFile returns an initialised empty lock file.
func NewLockFile() *LockFile {
	return &LockFile{
		Version: 1,
		Entries: make(map[string]LockEntry),
	}
}

// LoadLock reads and parses a .cops.lock file.
// Returns an empty lock file if the file does not exist.
func LoadLock(path string) (*LockFile, error) {
	lf := NewLockFile()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return lf, nil
		}
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	if err := json.Unmarshal(data, lf); err != nil {
		return nil, fmt.Errorf("parsing lock file: %w", err)
	}

	if lf.Entries == nil {
		lf.Entries = make(map[string]LockEntry)
	}

	return lf, nil
}

// Save writes the lock file to the given path.
func (lf *LockFile) Save(path string) error {
	data, err := json.MarshalIndent(lf, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding lock file: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}

	return nil
}

// entryKey builds the map key for a lock entry.
func entryKey(assetType, name string) string {
	return assetType + "/" + name
}

// Set records or updates a lock entry after a successful sync.
func (lf *LockFile) Set(assetType, name, ref, resolvedSHA, targetPath string, content []byte) {
	key := entryKey(assetType, name)
	lf.Entries[key] = LockEntry{
		Type:        assetType,
		Name:        name,
		Ref:         ref,
		ResolvedSHA: resolvedSHA,
		TargetPath:  targetPath,
		Checksum:    checksum(content),
		SyncedAt:    time.Now().UTC().Format(time.RFC3339),
	}
}

// Get retrieves a lock entry, if it exists.
func (lf *LockFile) Get(assetType, name string) (LockEntry, bool) {
	key := entryKey(assetType, name)
	e, ok := lf.Entries[key]
	return e, ok
}

// Remove deletes a lock entry.
func (lf *LockFile) Remove(assetType, name string) {
	key := entryKey(assetType, name)
	delete(lf.Entries, key)
}

// checksum returns the hex-encoded SHA-256 of the given data.
func checksum(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
