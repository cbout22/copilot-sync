package usecase

import (
	"strings"
	"testing"
)

func TestUseAsset_Execute_Success(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	gh := &mockGitHub{
		files: map[string][]byte{
			"myorg/myrepo/agents/helper@v2.0": []byte("# Agent content"),
		},
		sha: "sha999",
	}

	uc := NewUseAsset(mfs, gh)
	result, err := uc.Execute("agents", "helper", "myorg/myrepo/agents/helper@v2.0", "/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if result.TargetPath != ".github/agents/helper.agent.md" {
		t.Errorf("TargetPath: got %q, want %q", result.TargetPath, ".github/agents/helper.agent.md")
	}

	// Verify file was written
	content, err := mfs.ReadFile("/proj/.github/agents/helper.agent.md")
	if err != nil {
		t.Fatalf("reading written file: %v", err)
	}
	if string(content) != "# Agent content" {
		t.Errorf("file content: got %q, want %q", content, "# Agent content")
	}

	// Verify manifest was written
	manifestData, err := mfs.ReadFile("/proj/copilot.toml")
	if err != nil {
		t.Fatalf("reading manifest: %v", err)
	}
	if !strings.Contains(string(manifestData), "helper") {
		t.Error("manifest does not contain the new entry")
	}

	// Verify lock file was written
	lockData, err := mfs.ReadFile("/proj/.cops.lock")
	if err != nil {
		t.Fatalf("reading lock: %v", err)
	}
	if !strings.Contains(string(lockData), "sha999") {
		t.Error("lock file does not contain the resolved SHA")
	}
}

func TestUseAsset_Execute_InvalidType(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	gh := &mockGitHub{sha: "abc"}
	uc := NewUseAsset(mfs, gh)

	_, err := uc.Execute("widgets", "thing", "org/repo/path@v1", "/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err == nil {
		t.Fatal("expected error for invalid type, got nil")
	}
}

func TestUseAsset_Execute_InvalidRef(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	gh := &mockGitHub{sha: "abc"}
	uc := NewUseAsset(mfs, gh)

	_, err := uc.Execute("instructions", "bad", "not-valid", "/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err == nil {
		t.Fatal("expected error for invalid ref, got nil")
	}
}
