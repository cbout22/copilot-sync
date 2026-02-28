package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- NewLockFile ---

func TestNewLockFile(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	if lf.Version != 1 {
		t.Errorf("Version = %d, want 1", lf.Version)
	}
	if lf.Entries == nil {
		t.Error("Entries is nil")
	}
	if len(lf.Entries) != 0 {
		t.Error("Entries not empty")
	}
}

// --- checksum ---

func TestChecksum_KnownValue(t *testing.T) {
	t.Parallel()
	// echo -n "hello" | sha256sum
	got := checksum([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("checksum(\"hello\") = %q, want %q", got, want)
	}
}

func TestChecksum_Deterministic(t *testing.T) {
	t.Parallel()
	data := []byte("some content here")
	c1 := checksum(data)
	for i := 0; i < 100; i++ {
		if got := checksum(data); got != c1 {
			t.Fatalf("iteration %d: checksum differs: %q vs %q", i, got, c1)
		}
	}
}

func TestChecksum_EmptyInput(t *testing.T) {
	t.Parallel()
	got := checksum([]byte{})
	want := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	if got != want {
		t.Errorf("checksum(empty) = %q, want %q", got, want)
	}
}

// --- entryKey ---

func TestEntryKey_Format(t *testing.T) {
	t.Parallel()
	cases := []struct{ typ, name, want string }{
		{"instructions", "my-review", "instructions/my-review"},
		{"agents", "helper", "agents/helper"},
		{"prompts", "deploy", "prompts/deploy"},
		{"skills", "web-tool", "skills/web-tool"},
	}
	for _, tc := range cases {
		if got := entryKey(tc.typ, tc.name); got != tc.want {
			t.Errorf("entryKey(%q, %q) = %q, want %q", tc.typ, tc.name, got, tc.want)
		}
	}
}

// --- Set ---

func TestLockFile_Set_FieldsPopulated(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	content := []byte("file content here")
	lf.Set("agents", "helper", "org/repo/a.md@v1", "sha789", ".github/agents/helper.agent.md", content)

	entry, ok := lf.Get("agents", "helper")
	if !ok {
		t.Fatal("entry not found")
	}
	if entry.Type != "agents" {
		t.Errorf("Type = %q, want %q", entry.Type, "agents")
	}
	if entry.Name != "helper" {
		t.Errorf("Name = %q, want %q", entry.Name, "helper")
	}
	if entry.Ref != "org/repo/a.md@v1" {
		t.Errorf("Ref = %q", entry.Ref)
	}
	if entry.ResolvedSHA != "sha789" {
		t.Errorf("ResolvedSHA = %q", entry.ResolvedSHA)
	}
	if entry.TargetPath != ".github/agents/helper.agent.md" {
		t.Errorf("TargetPath = %q", entry.TargetPath)
	}

	expectedChecksum := checksum(content)
	if entry.Checksum != expectedChecksum {
		t.Errorf("Checksum = %q, want %q", entry.Checksum, expectedChecksum)
	}

	if _, err := time.Parse(time.RFC3339, entry.SyncedAt); err != nil {
		t.Errorf("SyncedAt %q is not valid RFC3339: %v", entry.SyncedAt, err)
	}
}

func TestLockFile_Set_KeyFormat(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Set("prompts", "deploy", "ref", "sha", "path", []byte{})
	if _, ok := lf.Entries["prompts/deploy"]; !ok {
		keys := make([]string, 0, len(lf.Entries))
		for k := range lf.Entries {
			keys = append(keys, k)
		}
		t.Errorf("expected key 'prompts/deploy', got keys: %v", keys)
	}
}

// --- Get ---

func TestLockFile_Get_Missing(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	_, ok := lf.Get("instructions", "ghost")
	if ok {
		t.Error("expected false for missing entry")
	}
}

// --- Remove ---

func TestLockFile_Remove(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Set("skills", "web-tool", "ref", "sha", "path", []byte{})
	lf.Remove("skills", "web-tool")
	if _, ok := lf.Get("skills", "web-tool"); ok {
		t.Error("entry still present after Remove")
	}
}

func TestLockFile_Remove_Nonexistent(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Remove("skills", "ghost") // must not panic
}

// --- LoadLock ---

func TestLoadLock_Missing_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "does-not-exist.lock")
	lf, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if lf.Version != 1 {
		t.Errorf("Version = %d, want 1", lf.Version)
	}
	if len(lf.Entries) != 0 {
		t.Error("expected empty entries")
	}
}

func TestLoadLock_ValidJSON(t *testing.T) {
	t.Parallel()
	raw := `{
  "version": 1,
  "entries": {
    "instructions/reviews": {
      "type": "instructions",
      "name": "reviews",
      "ref": "github/awesome-copilot/instructions/code-review-generic.instructions.md@latest",
      "resolved_sha": "dc8b0cc5466dcaa482b1bb8b13b529bf8031d25a",
      "target_path": ".github/instructions/reviews.instructions.md",
      "checksum": "e6917309e3e986cde1650768cc871f54e5d054e90c4518afb5c937941ee212a9",
      "synced_at": "2026-02-23T07:56:57Z"
    }
  }
}`
	path := filepath.Join(t.TempDir(), ".cops.lock")
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	lf, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if lf.Version != 1 {
		t.Errorf("Version = %d", lf.Version)
	}
	entry, ok := lf.Get("instructions", "reviews")
	if !ok {
		t.Fatal("entry not found")
	}
	if entry.ResolvedSHA != "dc8b0cc5466dcaa482b1bb8b13b529bf8031d25a" {
		t.Errorf("ResolvedSHA = %q", entry.ResolvedSHA)
	}
	if entry.SyncedAt != "2026-02-23T07:56:57Z" {
		t.Errorf("SyncedAt = %q", entry.SyncedAt)
	}
	if entry.Checksum != "e6917309e3e986cde1650768cc871f54e5d054e90c4518afb5c937941ee212a9" {
		t.Errorf("Checksum = %q", entry.Checksum)
	}
}

func TestLoadLock_InvalidJSON(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "bad.lock")
	if err := os.WriteFile(path, []byte("{invalid json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadLock(path)
	if err == nil {
		t.Error("expected parse error for invalid JSON")
	}
}

// --- Save + Roundtrip ---

func TestLockFile_Save_Roundtrip(t *testing.T) {
	t.Parallel()
	lf1 := NewLockFile()
	lf1.Entries["agents/helper"] = LockEntry{
		Type:        "agents",
		Name:        "helper",
		Ref:         "org/repo/a.md@v1",
		ResolvedSHA: "sha123",
		TargetPath:  ".github/agents/helper.agent.md",
		Checksum:    "abc",
		SyncedAt:    "2024-01-01T00:00:00Z",
	}

	path := filepath.Join(t.TempDir(), ".cops.lock")
	if err := lf1.Save(path); err != nil {
		t.Fatal(err)
	}

	lf2, err := LoadLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if lf2.Version != 1 {
		t.Errorf("Version = %d after roundtrip", lf2.Version)
	}
	entry, ok := lf2.Get("agents", "helper")
	if !ok {
		t.Fatal("entry not found after roundtrip")
	}
	if entry.Checksum != "abc" {
		t.Errorf("Checksum = %q after roundtrip", entry.Checksum)
	}
	if entry.SyncedAt != "2024-01-01T00:00:00Z" {
		t.Errorf("SyncedAt = %q after roundtrip", entry.SyncedAt)
	}
	if entry.ResolvedSHA != "sha123" {
		t.Errorf("ResolvedSHA = %q after roundtrip", entry.ResolvedSHA)
	}
}

// --- Determinism ---

func TestLockFile_Save_DeterministicJSON(t *testing.T) {
	t.Parallel()
	newLF := func() *LockFile {
		lf := NewLockFile()
		// Add entries with keys in reverse order to exercise map key sorting
		lf.Entries["instructions/z-review"] = LockEntry{
			Type: "instructions", Name: "z-review",
			Ref: "ref-z", ResolvedSHA: "sha-z",
			TargetPath: ".github/instructions/z-review.instructions.md",
			Checksum: "chk-z", SyncedAt: "2024-01-01T00:00:00Z",
		}
		lf.Entries["instructions/a-review"] = LockEntry{
			Type: "instructions", Name: "a-review",
			Ref: "ref-a", ResolvedSHA: "sha-a",
			TargetPath: ".github/instructions/a-review.instructions.md",
			Checksum: "chk-a", SyncedAt: "2024-01-01T00:00:00Z",
		}
		return lf
	}

	path1 := filepath.Join(t.TempDir(), "first.lock")
	path2 := filepath.Join(t.TempDir(), "second.lock")
	if err := newLF().Save(path1); err != nil {
		t.Fatal(err)
	}
	if err := newLF().Save(path2); err != nil {
		t.Fatal(err)
	}

	b1, _ := os.ReadFile(path1)
	b2, _ := os.ReadFile(path2)
	if !bytes.Equal(b1, b2) {
		t.Errorf("non-deterministic JSON:\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}

func TestLockFile_Save_DeterministicJSON_MultipleRuns(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Entries["skills/z-skill"] = LockEntry{
		Type: "skills", Name: "z-skill", SyncedAt: "2024-01-01T00:00:00Z",
	}
	lf.Entries["agents/a-agent"] = LockEntry{
		Type: "agents", Name: "a-agent", SyncedAt: "2024-01-01T00:00:00Z",
	}
	lf.Entries["instructions/m-inst"] = LockEntry{
		Type: "instructions", Name: "m-inst", SyncedAt: "2024-01-01T00:00:00Z",
	}

	var reference []byte
	for i := 0; i < 20; i++ {
		path := filepath.Join(t.TempDir(), "run.lock")
		if err := lf.Save(path); err != nil {
			t.Fatal(err)
		}
		data, _ := os.ReadFile(path)
		if i == 0 {
			reference = data
		} else if !bytes.Equal(data, reference) {
			t.Fatalf("iteration %d: JSON output differs:\ngot:\n%s\nwant:\n%s", i, data, reference)
		}
	}
}

func TestLockFile_Save_GoldenJSON(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Entries["instructions/reviews"] = LockEntry{
		Type:        "instructions",
		Name:        "reviews",
		Ref:         "org/repo/f.md@v1",
		ResolvedSHA: "abc",
		TargetPath:  ".github/instructions/reviews.instructions.md",
		Checksum:    "def",
		SyncedAt:    "2024-01-01T00:00:00Z",
	}

	path := filepath.Join(t.TempDir(), ".cops.lock")
	if err := lf.Save(path); err != nil {
		t.Fatal(err)
	}
	got := string(readBytesLock(t, path))

	want := `{
  "version": 1,
  "entries": {
    "instructions/reviews": {
      "type": "instructions",
      "name": "reviews",
      "ref": "org/repo/f.md@v1",
      "resolved_sha": "abc",
      "target_path": ".github/instructions/reviews.instructions.md",
      "checksum": "def",
      "synced_at": "2024-01-01T00:00:00Z"
    }
  }
}`
	if got != want {
		t.Errorf("golden JSON mismatch:\ngot:\n%s\nwant:\n%s", got, want)
	}
}

func TestLockFile_Save_MultipleEntries_KeysAreSorted(t *testing.T) {
	t.Parallel()
	lf := NewLockFile()
	lf.Entries["skills/z-skill"] = LockEntry{
		Type: "skills", Name: "z-skill", SyncedAt: "2024-01-01T00:00:00Z",
	}
	lf.Entries["agents/a-agent"] = LockEntry{
		Type: "agents", Name: "a-agent", SyncedAt: "2024-01-01T00:00:00Z",
	}
	lf.Entries["instructions/m-inst"] = LockEntry{
		Type: "instructions", Name: "m-inst", SyncedAt: "2024-01-01T00:00:00Z",
	}

	path := filepath.Join(t.TempDir(), ".cops.lock")
	if err := lf.Save(path); err != nil {
		t.Fatal(err)
	}
	got := string(readBytesLock(t, path))

	// JSON must list keys in alphabetical order
	posAgent := strings.Index(got, "agents/a-agent")
	posInst := strings.Index(got, "instructions/m-inst")
	posSkill := strings.Index(got, "skills/z-skill")
	if posAgent < 0 || posInst < 0 || posSkill < 0 {
		t.Fatalf("keys not found in output:\n%s", got)
	}
	if !(posAgent < posInst && posInst < posSkill) {
		t.Errorf("JSON keys not in sorted order: agents=%d, instructions=%d, skills=%d\noutput:\n%s",
			posAgent, posInst, posSkill, got)
	}
}

// readBytesLock is a local helper (avoids collision with manifest_test.go helpers
// since both are in package manifest).
func readBytesLock(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
