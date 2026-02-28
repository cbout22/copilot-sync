package injector

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/resolver"
)

// Injector downloads assets from GitHub and writes them to the correct
// .github/<type>/ directory.
type Injector struct {
	source  resolver.SourceRepository
	lock    *manifest.LockFile
	rootDir string // project root directory
}

// New creates an Injector.
func New(source resolver.SourceRepository, lock *manifest.LockFile, rootDir string) *Injector {
	return &Injector{
		source:  source,
		lock:    lock,
		rootDir: rootDir,
	}
}

// InjectResult holds the outcome of injecting a single asset.
type InjectResult struct {
	Type       string
	Name       string
	Ref        string
	TargetPath string
	SHA        string
	Err        error
}

// Inject downloads and writes a single asset.
func (inj *Injector) Inject(assetType config.AssetType, name, rawRef string) InjectResult {
	result := InjectResult{
		Type: string(assetType),
		Name: name,
		Ref:  rawRef,
	}

	// Parse the reference
	ref, err := config.ParseRef(rawRef)
	if err != nil {
		result.Err = err
		return result
	}

	targetPath := assetType.TargetPath(name)
	result.TargetPath = targetPath
	absTarget := filepath.Join(inj.rootDir, targetPath)

	if assetType.IsDirectory() {
		err = inj.injectDirectory(ref, absTarget)
	} else {
		err = inj.injectFile(ref, absTarget, assetType, name, rawRef)
	}

	result.Err = err
	return result
}

// injectFile downloads a single file asset and writes it to disk.
func (inj *Injector) injectFile(ref config.AssetRef, absTarget string, assetType config.AssetType, name, rawRef string) error {
	// Ensure target directory exists
	if err := os.MkdirAll(filepath.Dir(absTarget), 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Resolve commit SHA for the lock file
	sha, err := inj.source.ResolveSHA(ref)
	if err != nil {
		return fmt.Errorf("resolving commit SHA: %w", err)
	}

	// Download the file
	content, err := inj.source.DownloadFile(ref)
	if err != nil {
		return err
	}

	// Remove existing file if it exists to avoid stale content
	if _, err := os.Stat(absTarget); err == nil {
		if err := os.Remove(absTarget); err != nil {
			return fmt.Errorf("removing existing file: %w", err)
		}
	}

	// Write to disk
	if err := os.WriteFile(absTarget, content, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", absTarget, err)
	}

	// Update the lock file
	inj.lock.Set(string(assetType), name, rawRef, sha, assetType.TargetPath(name), content)

	return nil
}

// computeDirectoryChecksum creates a combined checksum for all files in a directory.
func computeDirectoryChecksum(contents map[string][]byte) []byte {
	// Sort keys so the checksum is deterministic regardless of map iteration order.
	keys := make([]string, 0, len(contents))
	for k := range contents {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var combined []byte
	for _, k := range keys {
		combined = append(combined, contents[k]...)
	}
	return combined
}

// injectDirectory downloads all files in a directory (for skills) and writes them.
func (inj *Injector) injectDirectory(ref config.AssetRef, absTargetDir string) error {
	// List all files in the remote directory
	entries, err := inj.source.ListDirectory(ref)
	if err != nil {
		return err
	}

	// Ensure base target directory exists
	if err := os.MkdirAll(absTargetDir, 0755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	// Track all downloaded contents for checksum
	allContents := make(map[string][]byte)

	for _, entry := range entries {
		// Compute relative path within the skill directory
		relPath := strings.TrimPrefix(entry.Path, ref.Path+"/")
		if relPath == entry.Path {
			// It's the directory entry itself, use the filename
			relPath = filepath.Base(entry.Path)
		}

		targetFile := filepath.Join(absTargetDir, relPath)

		// Ensure subdirectories exist
		if err := os.MkdirAll(filepath.Dir(targetFile), 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", relPath, err)
		}

		// Download each file using raw URL
		fileRef := config.AssetRef{
			Org:  ref.Org,
			Repo: ref.Repo,
			Path: entry.Path,
			Ref:  ref.Ref,
		}

		content, err := inj.source.DownloadFile(fileRef)
		if err != nil {
			return fmt.Errorf("downloading %s: %w", entry.Path, err)
		}

		if err := os.WriteFile(targetFile, content, 0644); err != nil {
			return fmt.Errorf("writing %s: %w", targetFile, err)
		}

		allContents[relPath] = content
	}

	// Resolve commit SHA for the lock file
	sha, err := inj.source.ResolveSHA(ref)
	if err != nil {
		// Non-fatal: we still wrote the files, just can't lock the SHA
		sha = "unknown"
	}

	// Update the lock file with combined checksum
	combinedContent := computeDirectoryChecksum(allContents)
	targetPath := strings.TrimPrefix(absTargetDir, inj.rootDir+"/")
	inj.lock.Set("skills", filepath.Base(absTargetDir), ref.Raw(), sha, targetPath, combinedContent)

	return nil
}
