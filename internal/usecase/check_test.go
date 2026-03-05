package usecase

import (
	"testing"
)

func TestCheckAssets_Execute_AllInSync(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	manifestContent := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	// Create file on disk
	_ = mfs.MkdirAll("/proj/.github/instructions", 0755)
	_ = mfs.WriteFile("/proj/.github/instructions/setup.instructions.md", []byte("content"), 0644)

	// Create matching lock
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
	_ = mfs.WriteFile("/proj/.cops.lock", []byte(lockContent), 0644)

	uc := NewCheckAssets(mfs)
	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if result.Issues() != 0 {
		t.Errorf("expected 0 issues, got %d", result.Issues())
	}

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Status != StatusOK {
		t.Errorf("expected StatusOK, got %d", result.Entries[0].Status)
	}
}

func TestCheckAssets_Execute_MissingNeverSynced(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	manifestContent := `[instructions]
setup = "myorg/myrepo/instructions/setup@v1.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	uc := NewCheckAssets(mfs)
	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if result.Issues() != 1 {
		t.Errorf("expected 1 issue, got %d", result.Issues())
	}
	if result.Entries[0].Status != StatusMissingNeverSynced {
		t.Errorf("expected StatusMissingNeverSynced, got %d", result.Entries[0].Status)
	}
}

func TestCheckAssets_Execute_RefChanged(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	manifestContent := `[instructions]
setup = "myorg/myrepo/instructions/setup@v2.0"
`
	_ = mfs.WriteFile("/proj/copilot.toml", []byte(manifestContent), 0644)

	// File exists
	_ = mfs.MkdirAll("/proj/.github/instructions", 0755)
	_ = mfs.WriteFile("/proj/.github/instructions/setup.instructions.md", []byte("old"), 0644)

	// Lock says v1.0
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
	_ = mfs.WriteFile("/proj/.cops.lock", []byte(lockContent), 0644)

	uc := NewCheckAssets(mfs)
	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if result.Issues() != 1 {
		t.Errorf("expected 1 issue, got %d", result.Issues())
	}
	if result.Entries[0].Status != StatusRefChanged {
		t.Errorf("expected StatusRefChanged, got %d", result.Entries[0].Status)
	}
}

func TestCheckAssets_Execute_EmptyManifest(t *testing.T) {
	t.Parallel()

	mfs := newMockFS()
	uc := NewCheckAssets(mfs)

	result, err := uc.Execute("/proj/copilot.toml", "/proj/.cops.lock", "/proj")
	if err != nil {
		t.Fatalf("Execute: unexpected error: %v", err)
	}

	if len(result.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(result.Entries))
	}
}
