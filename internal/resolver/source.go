package resolver

import "github.com/cbout22/copilot-sync/internal/config"

// SourceRepository defines operations for fetching assets from a remote source.
type SourceRepository interface {
	// DownloadFile fetches a single file's content by reference.
	DownloadFile(ref config.AssetRef) ([]byte, error)

	// ListDirectory returns all file entries under a directory reference.
	// Used for skills which are downloaded as entire directories.
	ListDirectory(ref config.AssetRef) ([]GitHubTreeEntry, error)

	// ResolveSHA resolves a ref (branch, tag, SHA) to a full commit SHA.
	ResolveSHA(ref config.AssetRef) (string, error)
}
