package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/cbout22/copilot-sync/internal/auth"
)

const githubAPIBase = "https://api.github.com"

// resolveGitHubCompletions provides dynamic shell completion for GitHub assets.
func resolveGitHubCompletions(toComplete string) ([]string, cobra.ShellCompDirective) {
	// Use a short timeout to prevent blocking the shell
	client, err := auth.NewHTTPClientWithTimeout(time.Second)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// 1. Version state: Typing `@`
	if idx := strings.Index(toComplete, "@"); idx != -1 {
		repoPart := toComplete[:idx]
		refPart := toComplete[idx+1:]
		return completeRefs(client, repoPart, refPart)
	}

	// 2. Path state: Typing `repo/` (after `org/repo/`)
	parts := strings.Split(toComplete, "/")
	if len(parts) >= 3 {
		org := parts[0]
		repo := parts[1]
		pathPrefix := strings.Join(parts[2:], "/")
		return completePaths(client, org, repo, pathPrefix, toComplete)
	}

	// 3. Org/Repo state: Typing `org/` or just `org`
	return completeRepos(client, toComplete)
}

func completeRepos(client *http.Client, toComplete string) ([]string, cobra.ShellCompDirective) {
	// If it's empty, we can't really search effectively without a query
	if toComplete == "" {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Search repositories
	// If user typed "org/", search for "user:org" or "org:org"
	// If user typed "org/re", search for "repo:org/re" or "org/re in:name"
	query := toComplete
	if strings.Contains(toComplete, "/") {
		parts := strings.SplitN(toComplete, "/", 2)
		org := parts[0]
		repoPrefix := parts[1]
		query = fmt.Sprintf("user:%s %s in:name", org, repoPrefix)
	}

	searchURL := fmt.Sprintf("%s/search/repositories?q=%s&per_page=10", githubAPIBase, url.QueryEscape(query))
	resp, err := client.Get(searchURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer resp.Body.Close()

	var result struct {
		Items []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
		} `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	for _, item := range result.Items {
		if strings.HasPrefix(item.FullName, toComplete) {
			desc := item.Description
			if desc == "" {
				desc = "Repository"
			}
			// Truncate description if too long
			if len(desc) > 60 {
				desc = desc[:57] + "..."
			}
			// Add trailing slash to encourage path completion
			completions = append(completions, formatCompletionLine(item.FullName, desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completePaths(client *http.Client, org, repo, pathPrefix, fullToComplete string) ([]string, cobra.ShellCompDirective) {
	// We need to get the default branch first to get the tree, or just use HEAD
	// Actually, we can use HEAD for the tree
	treeURL := fmt.Sprintf("%s/repos/%s/%s/git/trees/HEAD?recursive=1", githubAPIBase, org, repo)
	resp, err := client.Get(treeURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	defer resp.Body.Close()

	var result struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"tree"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	var completions []string
	basePrefix := fmt.Sprintf("%s/%s/", org, repo)

	// We only want to suggest the next level of directories or matching files
	// Since we have recursive=1, we get all paths.
	// We should filter paths that start with pathPrefix

	seenDirs := make(map[string]bool)

	for _, item := range result.Tree {
		if !strings.HasPrefix(item.Path, pathPrefix) {
			continue
		}

		// If it's a file, only suggest .md files
		if item.Type == "blob" && !strings.HasSuffix(item.Path, ".md") {
			continue
		}

		// We want to suggest the immediate next segment
		remainingPath := strings.TrimPrefix(item.Path, pathPrefix)
		segments := strings.Split(remainingPath, "/")

		if len(segments) > 1 {
			// It's a directory further down
			nextDir := pathPrefix + segments[0] + "/"
			if !seenDirs[nextDir] {
				seenDirs[nextDir] = true
				completions = append(completions, formatCompletionLine(basePrefix+nextDir, "Directory"))
			}
		} else {
			// It's an immediate file or directory
			if item.Type == "tree" {
				dirPath := item.Path + "/"
				if !seenDirs[dirPath] {
					seenDirs[dirPath] = true
					completions = append(completions, formatCompletionLine(basePrefix+dirPath, "Directory"))
				}
			} else {
				completions = append(completions, formatCompletionLine(basePrefix+item.Path, "Markdown File"))
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func completeRefs(client *http.Client, repoPart, refPart string) ([]string, cobra.ShellCompDirective) {
	parts := strings.Split(repoPart, "/")
	if len(parts) < 3 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	org := parts[0]
	repo := parts[1]

	var completions []string

	// Always suggest latest if it matches
	if strings.HasPrefix("latest", refPart) {
		completions = append(completions, formatCompletionLine(fmt.Sprintf("%s@latest", repoPart), "Default branch"))
	}

	// Fetch refs
	refsURL := fmt.Sprintf("%s/repos/%s/%s/git/refs", githubAPIBase, org, repo)
	resp, err := client.Get(refsURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
	defer resp.Body.Close()

	var result []struct {
		Ref string `json:"ref"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	for _, item := range result {
		// Only suggest branches and tags
		if !strings.HasPrefix(item.Ref, "refs/heads/") && !strings.HasPrefix(item.Ref, "refs/tags/") {
			continue
		}

		// refs/heads/main -> main
		// refs/tags/v1.0 -> v1.0
		shortRef := strings.TrimPrefix(item.Ref, "refs/heads/")
		shortRef = strings.TrimPrefix(shortRef, "refs/tags/")

		if strings.HasPrefix(shortRef, refPart) {
			desc := "Branch"
			if strings.HasPrefix(item.Ref, "refs/tags/") {
				desc = "Tag"
			}
			completions = append(completions, formatCompletionLine(fmt.Sprintf("%s@%s", repoPart, shortRef), desc))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
