package port

import "github.com/cbout22/copilot-sync/internal/config"

// GitHubResolver abstracts GitHub API operations.
type GitHubResolver interface {
	ResolveRef(ref config.AssetRef) (config.AssetRef, error)
	DownloadFile(ref config.AssetRef) ([]byte, error)
	ListDirectory(ref config.AssetRef) ([]config.GitHubTreeEntry, error)
	ResolveSHA(ref config.AssetRef) (string, error)
}
