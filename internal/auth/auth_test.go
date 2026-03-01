package auth

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestToken_GITHUB_TOKEN(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "gh-token-123")
	t.Setenv("GH_TOKEN", "")

	tok, err := Token()
	if err != nil {
		t.Fatalf("Token(): unexpected error: %v", err)
	}
	if tok != "gh-token-123" {
		t.Errorf("Token(): got %q, want %q", tok, "gh-token-123")
	}
}

func TestToken_GH_TOKEN_Fallback(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "fallback-token")

	tok, err := Token()
	if err != nil {
		t.Fatalf("Token(): unexpected error: %v", err)
	}
	if tok != "fallback-token" {
		t.Errorf("Token(): got %q, want %q", tok, "fallback-token")
	}
}

func TestToken_GITHUB_TOKEN_Priority(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "primary")
	t.Setenv("GH_TOKEN", "secondary")

	tok, err := Token()
	if err != nil {
		t.Fatalf("Token(): unexpected error: %v", err)
	}
	if tok != "primary" {
		t.Errorf("Token(): GITHUB_TOKEN should take priority, got %q", tok)
	}
}

func TestToken_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	_, err := Token()
	if err == nil {
		t.Fatal("Token(): expected error when no token set, got nil")
	}
}

func TestNewHTTPClient_WithToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-token")

	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient(): unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("NewHTTPClient(): returned nil client")
	}

	// Verify the token is sent in requests
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-token" {
			t.Errorf("Authorization header: got %q, want %q", auth, "Bearer test-token")
		}
		accept := r.Header.Get("Accept")
		if accept != "application/vnd.github.v3+json" {
			t.Errorf("Accept header: got %q, want %q", accept, "application/vnd.github.v3+json")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("client.Get(): %v", err)
	}
	resp.Body.Close()
}

func TestNewHTTPClient_NoToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")

	// Redirect stderr to avoid noise
	oldStderr := os.Stderr
	os.Stderr, _ = os.Open(os.DevNull)
	defer func() { os.Stderr = oldStderr }()

	client, err := NewHTTPClient()
	if err != nil {
		t.Fatalf("NewHTTPClient(): unexpected error: %v", err)
	}
	if client == nil {
		t.Fatal("NewHTTPClient(): returned nil client (should return unauthenticated client)")
	}
}

func TestNewHTTPClientWithTimeout(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "tok")

	client, err := NewHTTPClientWithTimeout(5 * time.Second)
	if err != nil {
		t.Fatalf("NewHTTPClientWithTimeout(): unexpected error: %v", err)
	}
	if client.Timeout != 5*time.Second {
		t.Errorf("Timeout: got %v, want %v", client.Timeout, 5*time.Second)
	}
}
