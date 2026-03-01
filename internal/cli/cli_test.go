package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/resolver"
)

// mockResolver implements resolver.ResolverAPI for testing without GitHub.
type mockResolver struct {
	files map[string][]byte // key: "org/repo/path@ref" → content
	sha   string
}

var _ resolver.ResolverAPI = (*mockResolver)(nil)

func (m *mockResolver) ResolveRef(ref config.AssetRef) (config.AssetRef, error) {
	return ref, nil
}

func (m *mockResolver) DownloadFile(ref config.AssetRef) ([]byte, error) {
	key := ref.Raw()
	if content, ok := m.files[key]; ok {
		return content, nil
	}
	return nil, os.ErrNotExist
}

func (m *mockResolver) ListDirectory(ref config.AssetRef) ([]resolver.GitHubTreeEntry, error) {
	return nil, os.ErrNotExist
}

func (m *mockResolver) ResolveSHA(ref config.AssetRef) (string, error) {
	return m.sha, nil
}

// setupTestDir creates a temp directory with an optional copilot.toml manifest.
func setupTestDir(t *testing.T, manifestContent string) (dir, manifestPath, lockPath string) {
	t.Helper()
	dir = t.TempDir()
	manifestPath = filepath.Join(dir, "copilot.toml")
	lockPath = filepath.Join(dir, ".cops.lock")
	if manifestContent != "" {
		if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir, manifestPath, lockPath
}

func TestSyncCmd_EmptyManifest(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")
	mock := &mockResolver{sha: "abc123"}

	err := runSyncWith(manifestPath, lockPath, mock, dir)
	if err != nil {
		t.Fatalf("runSyncWith(empty): unexpected error: %v", err)
	}
}

func TestSyncCmd_Success(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)
	fileContent := []byte("# Setup Instructions\nDo the thing.\n")

	mock := &mockResolver{
		files: map[string][]byte{
			"myorg/myrepo/instructions/setup@v1.0": fileContent,
		},
		sha: "abc123def",
	}

	err := runSyncWith(manifestPath, lockPath, mock, dir)
	if err != nil {
		t.Fatalf("runSyncWith: unexpected error: %v", err)
	}

	// Verify the file was written
	targetPath := filepath.Join(dir, ".github", "instructions", "setup.instructions.md")
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("reading synced file: %v", err)
	}
	if string(got) != string(fileContent) {
		t.Errorf("synced content: got %q, want %q", got, fileContent)
	}

	// Verify lock file was created
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	if !strings.Contains(string(lockData), "abc123def") {
		t.Error("lock file does not contain the resolved SHA")
	}
}

func TestUseCmd_AddsEntry(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")
	fileContent := []byte("# My Agent\nI help with testing.\n")

	mock := &mockResolver{
		files: map[string][]byte{
			"myorg/myrepo/agents/helper@v2.0": fileContent,
		},
		sha: "sha999",
	}

	err := runUseWith("agents", "helper", "myorg/myrepo/agents/helper@v2.0", manifestPath, lockPath, mock, dir)
	if err != nil {
		t.Fatalf("runUseWith: unexpected error: %v", err)
	}

	// Verify the file was written
	targetPath := filepath.Join(dir, ".github", "agents", "helper.agent.md")
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("reading injected file: %v", err)
	}
	if string(got) != string(fileContent) {
		t.Errorf("injected content: got %q, want %q", got, fileContent)
	}

	// Verify manifest was updated
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if !strings.Contains(string(manifestData), "helper") {
		t.Error("manifest does not contain the new entry")
	}

	// Verify lock file was updated
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("reading lock file: %v", err)
	}
	if !strings.Contains(string(lockData), "sha999") {
		t.Error("lock file does not contain the resolved SHA")
	}
}

func TestUseCmd_InvalidRef(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")
	mock := &mockResolver{sha: "abc"}

	err := runUseWith("instructions", "bad", "not-a-valid-ref", manifestPath, lockPath, mock, dir)
	if err == nil {
		t.Fatal("runUseWith(invalid ref): expected error, got nil")
	}
}

func TestUseCmd_InvalidType(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")
	mock := &mockResolver{sha: "abc"}

	err := runUseWith("widgets", "thing", "org/repo/path@v1", manifestPath, lockPath, mock, dir)
	if err == nil {
		t.Fatal("runUseWith(invalid type): expected error, got nil")
	}
}

func TestUnuseCmd_RemovesEntry(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)

	// Create the file on disk so unuse can delete it
	targetDir := filepath.Join(dir, ".github", "instructions")
	os.MkdirAll(targetDir, 0755)
	targetPath := filepath.Join(targetDir, "setup.instructions.md")
	os.WriteFile(targetPath, []byte("content"), 0644)

	err := runUnuseWith("instructions", "setup", manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("runUnuseWith: unexpected error: %v", err)
	}

	// Verify file was deleted
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("target file should have been deleted")
	}

	// Verify manifest entry was removed
	manifestData, _ := os.ReadFile(manifestPath)
	if strings.Contains(string(manifestData), "setup") {
		t.Error("manifest still contains the removed entry")
	}
}

func TestUnuseCmd_NotFound(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")

	err := runUnuseWith("instructions", "nonexistent", manifestPath, lockPath, dir)
	if err == nil {
		t.Fatal("runUnuseWith(not found): expected error, got nil")
	}
}

func TestCheckCmd_AllInSync(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)

	// Create the file on disk
	targetDir := filepath.Join(dir, ".github", "instructions")
	os.MkdirAll(targetDir, 0755)
	targetPath := filepath.Join(targetDir, "setup.instructions.md")
	os.WriteFile(targetPath, []byte("content"), 0644)

	// Create a matching lock file
	lockContent := `{
  "version": 1,
  "entries": {
    "instructions/setup": {
      "type": "instructions",
      "name": "setup",
      "ref": "myorg/myrepo/instructions/setup@v1.0",
      "resolved_sha": "abc123",
      "target_path": ".github/instructions/setup.instructions.md",
      "checksum": "abc",
      "synced_at": "2025-01-01T00:00:00Z"
    }
  }
}`
	os.WriteFile(lockPath, []byte(lockContent), 0644)

	err := runCheckWith(false, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("runCheckWith(in sync): unexpected error: %v", err)
	}
}

func TestCheckCmd_MissingFile(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)

	// No file on disk, no lock — should report "missing (never synced)"
	// Non-strict: returns nil but prints warning
	err := runCheckWith(false, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("runCheckWith(missing, non-strict): unexpected error: %v", err)
	}
}

func TestCheckCmd_Strict_MissingFile(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)

	// Strict mode: should return error when file is missing
	err := runCheckWith(true, manifestPath, lockPath, dir)
	if err == nil {
		t.Fatal("runCheckWith(strict, missing): expected error, got nil")
	}
}

func TestCheckCmd_EmptyManifest(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")

	err := runCheckWith(false, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("runCheckWith(empty): unexpected error: %v", err)
	}
}

func TestCheckCmd_RefChanged(t *testing.T) {
	t.Parallel()

	manifest := `[instructions]
setup = "myorg/myrepo/instructions/setup@v2.0"
`
	dir, manifestPath, lockPath := setupTestDir(t, manifest)

	// File exists
	targetDir := filepath.Join(dir, ".github", "instructions")
	os.MkdirAll(targetDir, 0755)
	os.WriteFile(filepath.Join(targetDir, "setup.instructions.md"), []byte("old"), 0644)

	// Lock says v1.0, manifest says v2.0 — should report ref changed
	lockContent := `{
  "version": 1,
  "entries": {
    "instructions/setup": {
      "type": "instructions",
      "name": "setup",
      "ref": "myorg/myrepo/instructions/setup@v1.0",
      "resolved_sha": "abc123",
      "target_path": ".github/instructions/setup.instructions.md",
      "checksum": "abc",
      "synced_at": "2025-01-01T00:00:00Z"
    }
  }
}`
	os.WriteFile(lockPath, []byte(lockContent), 0644)

	// Non-strict: should succeed (just warns)
	err := runCheckWith(false, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("runCheckWith(ref changed, non-strict): unexpected error: %v", err)
	}

	// Strict: should fail
	err = runCheckWith(true, manifestPath, lockPath, dir)
	if err == nil {
		t.Fatal("runCheckWith(ref changed, strict): expected error, got nil")
	}
}

// TestFullWorkflow_UseCheckSync tests the full use → check → sync → check lifecycle.
func TestFullWorkflow_UseCheckSync(t *testing.T) {
	t.Parallel()

	dir, manifestPath, lockPath := setupTestDir(t, "")
	fileContent := []byte("# Prompt content\nBe helpful.\n")

	mock := &mockResolver{
		files: map[string][]byte{
			"myorg/myrepo/prompts/helpful@v1.0": fileContent,
		},
		sha: "workflow123",
	}

	// Step 1: use — add an asset
	err := runUseWith("prompts", "helpful", "myorg/myrepo/prompts/helpful@v1.0", manifestPath, lockPath, mock, dir)
	if err != nil {
		t.Fatalf("use: %v", err)
	}

	// Step 2: check — should be in sync
	err = runCheckWith(true, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("check after use: %v", err)
	}

	// Step 3: sync — should succeed (already in sync)
	err = runSyncWith(manifestPath, lockPath, mock, dir)
	if err != nil {
		t.Fatalf("sync: %v", err)
	}

	// Step 4: check strict — should still be in sync
	err = runCheckWith(true, manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("check after sync: %v", err)
	}

	// Step 5: unuse — remove the asset
	err = runUnuseWith("prompts", "helpful", manifestPath, lockPath, dir)
	if err != nil {
		t.Fatalf("unuse: %v", err)
	}

	// Step 6: verify file is gone
	targetPath := filepath.Join(dir, ".github", "prompts", "helpful.prompt.md")
	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		t.Error("file should be deleted after unuse")
	}
}
