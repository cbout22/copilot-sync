package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// AssetType represents the kind of copilot asset being managed.
type AssetType string

const (
	Instructions AssetType = "instructions"
	Agents       AssetType = "agents"
	Prompts      AssetType = "prompts"
	Skills       AssetType = "skills"
)

// ValidAssetTypes returns all supported asset types.
func ValidAssetTypes() []AssetType {
	return []AssetType{Instructions, Agents, Prompts, Skills}
}

// IsValid checks whether the asset type is one of the known types.
func (t AssetType) IsValid() bool {
	switch t {
	case Instructions, Agents, Prompts, Skills:
		return true
	}
	return false
}

// FileExtension returns the file suffix used for this type.
// Skills are directories, so they return an empty string.
func (t AssetType) FileExtension() string {
	switch t {
	case Instructions:
		return ".instructions.md"
	case Agents:
		return ".agent.md"
	case Prompts:
		return ".prompt.md"
	case Skills:
		return "" // skills are directories
	}
	return ""
}

// TargetDir returns the .github subdirectory where assets of this type live.
func (t AssetType) TargetDir() string {
	return filepath.Join(".github", string(t))
}

// TargetPath returns the full relative path for a named asset.
// For skills this returns a directory path; for others a file path.
func (t AssetType) TargetPath(name string) string {
	if t == Skills {
		return filepath.Join(t.TargetDir(), name)
	}
	return filepath.Join(t.TargetDir(), name+t.FileExtension())
}

// IsDirectory returns true if this asset type maps to a folder (skills).
func (t AssetType) IsDirectory() bool {
	return t == Skills
}

// AssetRef represents a parsed reference like "org/repo/path/to/file@v1.2".
type AssetRef struct {
	Org  string // GitHub organisation or user
	Repo string // Repository name
	Path string // Path inside the repository
	Ref  string // Git ref: tag, branch, or commit SHA
}

// ParseRef parses a raw reference string into an AssetRef.
// Expected format: "org/repo/path/to/file@ref"
func ParseRef(raw string) (AssetRef, error) {
	// Split on @ to separate the ref
	parts := strings.SplitN(raw, "@", 2)
	if len(parts) != 2 || parts[1] == "" {
		return AssetRef{}, fmt.Errorf("invalid reference %q: must contain @<ref> (e.g. org/repo/path@v1.0)", raw)
	}

	ref := parts[1]
	pathPart := parts[0]

	// We need at least org/repo/path (3 segments minimum)
	segments := strings.SplitN(pathPart, "/", 3)
	if len(segments) < 3 || segments[0] == "" || segments[1] == "" || segments[2] == "" {
		return AssetRef{}, fmt.Errorf("invalid reference %q: must be org/repo/path@ref", raw)
	}

	return AssetRef{
		Org:  segments[0],
		Repo: segments[1],
		Path: segments[2],
		Ref:  ref,
	}, nil
}

// Raw returns the canonical string representation of the ref.
func (r AssetRef) Raw() string {
	return fmt.Sprintf("%s/%s/%s@%s", r.Org, r.Repo, r.Path, r.Ref)
}

// RepoFullName returns "org/repo".
func (r AssetRef) RepoFullName() string {
	return fmt.Sprintf("%s/%s", r.Org, r.Repo)
}
