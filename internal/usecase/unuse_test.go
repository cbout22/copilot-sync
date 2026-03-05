package usecase

import (
	"strings"
	"testing"
)

func TestUnuseAsset_Execute_Success(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()

	// Set up a manifest with an entry
	manifestContent := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	// Create the file on disk
	_ = mfs.MkdirAll("/proj/.github/instructions", 0755)
	_ = mfs.WriteFile("/proj/.github/instructions/setup.instructions.md", []byte("content"), 0644)

	uc := NewUnuseAsset(mfs)
	err := uc.Execute("instructions", "setup", "/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	// Verify file was deleted
	if _, err := mfs.Stat("/proj/.github/instructions/setup.instructions.md"); err == nil {
		t.Error("file should have been deleted")
	}

	// Verify manifest entry was removed
	manifestData, _ := mfs.ReadFile("/proj/copilot.toml")
	if strings.Contains(string(manifestData), "setup") {
		t.Error("manifest still contains the removed entry")
	}
}

func TestUnuseAsset_Execute_NotFound(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	uc := NewUnuseAsset(mfs)

	err := uc.Execute("instructions", "nonexistent", "/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err == nil {
		t.Fatal("expected error for missing entry, got nil")
	}
}
