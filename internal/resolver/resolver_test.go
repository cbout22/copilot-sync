package resolver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cbout22/copilot-sync/internal/config"
)

// newTestServer creates an httptest.Server with route handling for GitHub API endpoints.
func newTestServer(t *testing.T, routes map[string]func(w http.ResponseWriter, r *http.Request)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler, ok := routes[r.URL.Path]; ok {
			handler(w, r)
			return
		}
		// Also check with query string
		if handler, ok := routes[r.URL.RequestURI()]; ok {
			handler(w, r)
			return
		}
		t.Logf("unhandled request: %s %s", r.Method, r.URL)
		http.NotFound(w, r)
	}))
}

// newTestResolver creates a Resolver that points at the given test server
// by overriding the base URLs.
func newTestResolver(t *testing.T, ts *httptest.Server) *Resolver {
	t.Helper()
	// We override the package-level constants indirectly by using a custom
	// resolver that rewrites URLs. Instead, we'll create a helper that
	// patches the resolver's URLs.
	res := New(ts.Client())
	return res
}

func TestResolveRef_Latest(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/repos/myorg/myrepo": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{
				"default_branch": "main",
			})
		},
	})
	defer ts.Close()

	// Override the API base to point at our test server
	origAPI := githubAPIBase
	// We can't modify const, so we use a workaround — test the underlying method
	res := New(ts.Client())

	// Test ResolveDefaultBranchName by hitting the test server
	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "path/file", Ref: "latest"}

	// We need to make the resolver talk to our test server.
	// Since githubAPIBase is a const, we'll test the method logic directly
	// by verifying the function works when it gets a proper HTTP response.
	_ = origAPI // acknowledging the const

	// Create a custom client that redirects GitHub API calls to our test server
	client := &http.Client{
		Transport: &rewriteTransport{
			base:       ts.Client().Transport,
			apiBase:    ts.URL,
			rawBase:    ts.URL,
			origAPI:    githubAPIBase,
			origRaw:    githubRawBase,
		},
	}
	res = New(client)

	resolved, err := res.ResolveRef(ref)
	if err != nil {
		t.Fatalf("ResolveRef(latest): unexpected error: %v", err)
	}
	if resolved.Ref != "main" {
		t.Errorf("ResolveRef(latest): got ref %q, want %q", resolved.Ref, "main")
	}
}

func TestResolveRef_Passthrough(t *testing.T) {
	t.Parallel()

	res := New(&http.Client{})
	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "path/file", Ref: "v1.2.3"}

	resolved, err := res.ResolveRef(ref)
	if err != nil {
		t.Fatalf("ResolveRef(v1.2.3): unexpected error: %v", err)
	}
	if resolved.Ref != "v1.2.3" {
		t.Errorf("ResolveRef(v1.2.3): got ref %q, want %q", resolved.Ref, "v1.2.3")
	}
}

func TestDownloadFile_Success(t *testing.T) {
	t.Parallel()

	want := []byte("# My Instruction\nDo the thing.\n")
	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/myorg/myrepo/v1.0/instructions/setup.md": func(w http.ResponseWriter, r *http.Request) {
			w.Write(want)
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "instructions/setup.md", Ref: "v1.0"}
	got, err := res.DownloadFile(ref)
	if err != nil {
		t.Fatalf("DownloadFile: unexpected error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("DownloadFile: got %q, want %q", got, want)
	}
}

func TestDownloadFile_FallbackMdExtension(t *testing.T) {
	t.Parallel()

	want := []byte("# Fallback content\n")
	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		// The exact path returns 404, but path.md succeeds
		"/myorg/myrepo/v1.0/instructions/setup": func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		},
		"/myorg/myrepo/v1.0/instructions/setup.md": func(w http.ResponseWriter, r *http.Request) {
			w.Write(want)
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "instructions/setup", Ref: "v1.0"}
	got, err := res.DownloadFile(ref)
	if err != nil {
		t.Fatalf("DownloadFile fallback: unexpected error: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("DownloadFile fallback: got %q, want %q", got, want)
	}
}

func TestDownloadFile_NotFound(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "nope/missing", Ref: "v1.0"}
	_, err := res.DownloadFile(ref)
	if err == nil {
		t.Fatal("DownloadFile(missing): expected error, got nil")
	}
}

func TestListDirectory_FiltersBlobs(t *testing.T) {
	t.Parallel()

	treeResp := GitHubTreeResponse{
		SHA: "abc123",
		Tree: []GitHubTreeEntry{
			{Path: "skills/my-skill/main.go", Type: "blob", SHA: "aaa"},
			{Path: "skills/my-skill/lib", Type: "tree", SHA: "bbb"},        // directory, should be filtered
			{Path: "skills/my-skill/lib/util.go", Type: "blob", SHA: "ccc"},
			{Path: "other/file.go", Type: "blob", SHA: "ddd"},              // outside path, should be filtered
		},
	}

	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/repos/myorg/myrepo/git/trees/v1.0?recursive=1": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(treeResp)
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "skills/my-skill", Ref: "v1.0"}
	entries, err := res.ListDirectory(ref)
	if err != nil {
		t.Fatalf("ListDirectory: unexpected error: %v", err)
	}

	// Should include the 2 blobs under skills/my-skill, not the tree or other/file.go
	if len(entries) != 2 {
		t.Fatalf("ListDirectory: got %d entries, want 2", len(entries))
	}

	paths := make(map[string]bool)
	for _, e := range entries {
		paths[e.Path] = true
	}
	if !paths["skills/my-skill/main.go"] {
		t.Error("ListDirectory: missing skills/my-skill/main.go")
	}
	if !paths["skills/my-skill/lib/util.go"] {
		t.Error("ListDirectory: missing skills/my-skill/lib/util.go")
	}
}

func TestListDirectory_NoFiles(t *testing.T) {
	t.Parallel()

	treeResp := GitHubTreeResponse{
		SHA:  "abc123",
		Tree: []GitHubTreeEntry{},
	}

	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/repos/myorg/myrepo/git/trees/v1.0?recursive=1": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(treeResp)
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "skills/empty", Ref: "v1.0"}
	_, err := res.ListDirectory(ref)
	if err == nil {
		t.Fatal("ListDirectory(empty): expected error, got nil")
	}
}

func TestResolveSHA_Success(t *testing.T) {
	t.Parallel()

	wantSHA := "abc123def456789"
	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/repos/myorg/myrepo/commits/v1.0": func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]string{"sha": wantSHA})
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "path/file", Ref: "v1.0"}
	got, err := res.ResolveSHA(ref)
	if err != nil {
		t.Fatalf("ResolveSHA: unexpected error: %v", err)
	}
	if got != wantSHA {
		t.Errorf("ResolveSHA: got %q, want %q", got, wantSHA)
	}
}

func TestResolveSHA_Error(t *testing.T) {
	t.Parallel()

	ts := newTestServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		"/repos/myorg/myrepo/commits/v1.0": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		},
	})
	defer ts.Close()

	client := &http.Client{
		Transport: &rewriteTransport{
			base:    ts.Client().Transport,
			apiBase: ts.URL,
			rawBase: ts.URL,
			origAPI: githubAPIBase,
			origRaw: githubRawBase,
		},
	}
	res := New(client)

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "path/file", Ref: "v1.0"}
	_, err := res.ResolveSHA(ref)
	if err == nil {
		t.Fatal("ResolveSHA(404): expected error, got nil")
	}
}

func TestRawFileURL(t *testing.T) {
	t.Parallel()

	ref := config.AssetRef{Org: "myorg", Repo: "myrepo", Path: "instructions/setup.md", Ref: "v1.0"}
	got := RawFileURL(ref)
	want := "https://raw.githubusercontent.com/myorg/myrepo/v1.0/instructions/setup.md"
	if got != want {
		t.Errorf("RawFileURL: got %q, want %q", got, want)
	}
}

// rewriteTransport rewrites GitHub API and raw content URLs to point at a test server.
type rewriteTransport struct {
	base    http.RoundTripper
	apiBase string
	rawBase string
	origAPI string
	origRaw string
}

func (t *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	// Rewrite GitHub API URLs to point at test server
	if len(url) > len(t.origAPI) && url[:len(t.origAPI)] == t.origAPI {
		newURL := t.apiBase + url[len(t.origAPI):]
		newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		newReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(newReq)
	}

	// Rewrite raw content URLs to point at test server
	if len(url) > len(t.origRaw) && url[:len(t.origRaw)] == t.origRaw {
		newURL := t.rawBase + url[len(t.origRaw):]
		newReq, _ := http.NewRequestWithContext(req.Context(), req.Method, newURL, req.Body)
		newReq.Header = req.Header
		return http.DefaultTransport.RoundTrip(newReq)
	}

	return t.base.RoundTrip(req)
}

// Verify Resolver implements ResolverAPI at compile time.
var _ ResolverAPI = (*Resolver)(nil)
