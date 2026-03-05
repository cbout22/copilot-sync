package usecase

import (
	"strings"
	"testing"
)

func TestSyncAssets_Execute_Success(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	manifestContent := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	gh := &mockGitHub{
		files: map[string][]byte{
			"myorg/myrepo/instructions/setup@v1.0": []byte("# Setup"),
		},
		sha: "abc123",
	}

	uc := NewSyncAssets(mfs, gh)
	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if len(result.Succeeded) != 1 {
		t.Fatalf("expected 1 succeeded, got %d", len(result.Succeeded))
	}
	if len(result.Failed) != 0 {
		t.Fatalf("expected 0 failed, got %d", len(result.Failed))
	}

	// Verify file was written
	content, err := mfs.ReadFile("/proj/.github/instructions/setup.instructions.md")
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(content) != "# Setup" {
		t.Errorf("file content: got %q, want %q", content, "# Setup")
	}

	// Verify lock was written
	lockData, err := mfs.ReadFile("/proj/.cops.lock")
	if err != nil {
		t.Fatalf("reading lock: %v", err)
	}
	if !strings.Contains(string(lockData), "abc123") {
		t.Error("lock file does not contain the resolved SHA")
	}
}

func TestSyncAssets_Execute_EmptyManifest(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	gh := &mockGitHub{sha: "abc"}
	uc := NewSyncAssets(mfs, gh)

	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if len(result.Succeeded)+len(result.Failed) != 0 {
		t.Error("expected empty result for empty manifest")
	}
}

func TestSyncAssets_Execute_PartialFailure(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	manifestContent := `[instructions]
good = "myorg/myrepo/instructions/good@v1.0"
bad = "myorg/myrepo/instructions/bad@v1.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	gh := &mockGitHub{
		files: map[string][]byte{
			"myorg/myrepo/instructions/good@v1.0": []byte("# Good"),
		},
		sha: "abc",
	}

	uc := NewSyncAssets(mfs, gh)
	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if len(result.Failed) != 1 {
		t.Errorf("expected 1 failed, got %d", len(result.Failed))
	}
	if len(result.Succeeded) != 1 {
		t.Errorf("expected 1 succeeded, got %d", len(result.Succeeded))
	}
}
