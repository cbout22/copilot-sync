package resolver

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cops/internal/config"
)

const githubAPIBase = "https://api.github.com"
const githubRawBase = "https://raw.githubusercontent.com"

// Resolver turns asset references into downloadable URLs and fetches content.
type Resolver struct {
	client *http.Client
}

// New creates a Resolver with the given (authenticated) HTTP client.
func New(client *http.Client) *Resolver {
	return &Resolver{client: client}
}

// ResolveRef resolves special ref aliases. If the ref is "latest", it queries
// the GitHub API for the repository's default branch and returns a new AssetRef
// with that branch as the ref. Otherwise returns the ref unchanged.
func (r *Resolver) ResolveRef(ref config.AssetRef) (config.AssetRef, error) {
	if ref.Ref != "latest" {
		return ref, nil
	}

	url := fmt.Sprintf("%s/repos/%s/%s", githubAPIBase, ref.Org, ref.Repo)
	resp, err := r.client.Get(url)
	if err != nil {
		return ref, fmt.Errorf("fetching repo info for %s: %w", ref.RepoFullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return ref, fmt.Errorf("fetching repo info for %s: HTTP %d — %s", ref.RepoFullName(), resp.StatusCode, string(body))
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return ref, fmt.Errorf("decoding repo info: %w", err)
	}

	if repoInfo.DefaultBranch == "" {
		return ref, fmt.Errorf("could not determine default branch for %s", ref.RepoFullName())
	}

	ref.Ref = repoInfo.DefaultBranch
	return ref, nil
}

// RawFileURL builds the raw.githubusercontent.com URL for a single file.
func RawFileURL(ref config.AssetRef) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", githubRawBase, ref.Org, ref.Repo, ref.Ref, ref.Path)
}

// DownloadFile fetches a single file from GitHub using the raw content URL.
// If the exact path returns a 404, it retries with common extensions (.md).
func (r *Resolver) DownloadFile(ref config.AssetRef) ([]byte, error) {
	// Resolve @latest to the default branch
	ref, err := r.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	// Try the exact path first, then fall back to common extensions
	pathsToTry := []string{ref.Path}
	if !strings.HasSuffix(ref.Path, ".md") {
		pathsToTry = append(pathsToTry, ref.Path+".md")
	}

	var lastErr error
	for _, path := range pathsToTry {
		candidate := ref
		candidate.Path = path
		url := RawFileURL(candidate)

		resp, err := r.client.Get(url)
		if err != nil {
			lastErr = fmt.Errorf("fetching %s: %w", url, err)
			continue
		}

		if resp.StatusCode == http.StatusNotFound {
			resp.Body.Close()
			lastErr = fmt.Errorf("fetching %s: HTTP 404", url)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("fetching %s: HTTP %d — %s", url, resp.StatusCode, string(body))
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("reading response from %s: %w", url, err)
		}

		return data, nil
	}

	return nil, lastErr
}

// GitHubTreeEntry represents one item in the GitHub Trees API response.
type GitHubTreeEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "blob" or "tree"
	SHA  string `json:"sha"`
}

// GitHubTreeResponse is the response from the GitHub Trees API.
type GitHubTreeResponse struct {
	SHA  string            `json:"sha"`
	Tree []GitHubTreeEntry `json:"tree"`
}

// ListDirectory fetches the recursive file listing of a directory in the repo.
// This is used for skills which are downloaded as entire folders.
func (r *Resolver) ListDirectory(ref config.AssetRef) ([]GitHubTreeEntry, error) {
	// Resolve @latest to the default branch
	ref, err := r.ResolveRef(ref)
	if err != nil {
		return nil, err
	}

	// First, get the tree SHA for the ref
	url := fmt.Sprintf("%s/repos/%s/%s/git/trees/%s?recursive=1",
		githubAPIBase, ref.Org, ref.Repo, ref.Ref)

	resp, err := r.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("listing tree for %s: %w", ref.RepoFullName(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("listing tree for %s: HTTP %d — %s", ref.RepoFullName(), resp.StatusCode, string(body))
	}

	var treeResp GitHubTreeResponse
	if err := json.NewDecoder(resp.Body).Decode(&treeResp); err != nil {
		return nil, fmt.Errorf("decoding tree response: %w", err)
	}

	// Filter entries that are under the requested path and are blobs (files)
	var entries []GitHubTreeEntry
	prefix := ref.Path + "/"
	for _, e := range treeResp.Tree {
		if e.Type == "blob" && (e.Path == ref.Path || (len(e.Path) > len(prefix) && e.Path[:len(prefix)] == prefix)) {
			entries = append(entries, e)
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no files found under %s in %s@%s", ref.Path, ref.RepoFullName(), ref.Ref)
	}

	return entries, nil
}

// ResolveCommitSHA resolves the given ref (branch, tag, or SHA) to a commit SHA.
func (r *Resolver) ResolveCommitSHA(ref config.AssetRef) (string, error) {
	// Resolve @latest to the default branch
	ref, err := r.ResolveRef(ref)
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/commits/%s",
		githubAPIBase, ref.Org, ref.Repo, ref.Ref)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	// Only fetch the SHA, not the full commit
	req.Header.Set("Accept", "application/vnd.github.v3.sha")

	resp, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolving commit SHA for %s@%s: %w", ref.RepoFullName(), ref.Ref, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("resolving commit SHA: HTTP %d — %s", resp.StatusCode, string(body))
	}

	sha, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading commit SHA: %w", err)
	}

	return string(sha), nil
}
