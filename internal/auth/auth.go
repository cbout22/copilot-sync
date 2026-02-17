package auth

import (
	"fmt"
	"net/http"
	"os"
)

// githubTokenEnvVars lists the environment variables checked for a GitHub token,
// in priority order.
var githubTokenEnvVars = []string{
	"GITHUB_TOKEN",
	"GH_TOKEN",
}

// Token returns the GitHub personal access token from the environment.
// It checks GITHUB_TOKEN first, then GH_TOKEN.
func Token() (string, error) {
	for _, env := range githubTokenEnvVars {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf(
		"no GitHub token found: set %s or %s in your environment",
		githubTokenEnvVars[0], githubTokenEnvVars[1],
	)
}

// NewHTTPClient returns an *http.Client suitable for GitHub API calls.
// If a GitHub token is available it adds Bearer auth on every request.
// Otherwise it returns a plain client (sufficient for public repos, but
// subject to stricter rate limits).
func NewHTTPClient() (*http.Client, error) {
	token, err := Token()
	if err != nil {
		// No token — return a plain client for public repo access
		fmt.Fprintf(os.Stderr, "⚠️  No GitHub token found — using unauthenticated requests (rate-limited).\n")
		fmt.Fprintf(os.Stderr, "   Set GITHUB_TOKEN or GH_TOKEN for private repos and higher rate limits.\n")
		return http.DefaultClient, nil
	}

	return &http.Client{
		Transport: &tokenTransport{
			token: token,
			base:  http.DefaultTransport,
		},
	}, nil
}

// tokenTransport is a custom http.RoundTripper that adds the Authorization header.
type tokenTransport struct {
	token string
	base  http.RoundTripper
}

func (t *tokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	r := req.Clone(req.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	r.Header.Set("Accept", "application/vnd.github.v3+json")
	return t.base.RoundTrip(r)
}
