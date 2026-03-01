package cli

import (
	"testing"

	"github.com/cbout22/copilot-sync/internal/manifest"
)

// testFileWriter is a minimal in-memory FileWriter for checker tests.
type testFileWriter struct {
	files map[string]bool // paths that "exist"
}

func newTestFileWriter(paths ...string) *testFileWriter {
	fw := &testFileWriter{files: make(map[string]bool)}
	for _, p := range paths {
		fw.files[p] = true
	}
	return fw
}

func (f *testFileWriter) Write(path string, data []byte) error { return nil }
func (f *testFileWriter) MkdirAll(path string) error           { return nil }
func (f *testFileWriter) Remove(path string) error             { return nil }
func (f *testFileWriter) Exists(path string) bool              { return f.files[path] }

func TestCheckAssets_AllSynced(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "helper", Ref: "org/repo/path@v1"},
		{Type: "instructions", Name: "review", Ref: "org/repo/review.md@v2"},
	}

	lock := manifest.NewLockFile()
	lock.Set("agents", "helper", "org/repo/path@v1", "sha1", ".github/agents/helper.agent.md", []byte("x"))
	lock.Set("instructions", "review", "org/repo/review.md@v2", "sha2", ".github/instructions/review.instructions.md", []byte("y"))

	fs := newTestFileWriter(
		".github/agents/helper.agent.md",
		".github/instructions/review.instructions.md",
	)

	results := CheckAssets(entries, lock, fs)

	if len(results) != 2 {
		t.Fatalf("got %d results, want 2", len(results))
	}
	for _, r := range results {
		if r.Status != CheckOK {
			t.Errorf("%s/%s: status = %d, want CheckOK", r.Type, r.Name, r.Status)
		}
	}
}

func TestCheckAssets_NeverSynced(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "helper", Ref: "org/repo/path@v1"},
	}
	lock := manifest.NewLockFile() // empty lock
	fs := newTestFileWriter()       // no files on disk

	results := CheckAssets(entries, lock, fs)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Status != CheckNeverSynced {
		t.Errorf("status = %d, want CheckNeverSynced", results[0].Status)
	}
}

func TestCheckAssets_FileMissing(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "helper", Ref: "org/repo/path@v1"},
	}
	lock := manifest.NewLockFile()
	lock.Set("agents", "helper", "org/repo/path@v1", "sha1", ".github/agents/helper.agent.md", []byte("x"))

	fs := newTestFileWriter() // file does NOT exist on disk

	results := CheckAssets(entries, lock, fs)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Status != CheckFileMissing {
		t.Errorf("status = %d, want CheckFileMissing", results[0].Status)
	}
}

func TestCheckAssets_NotInLock(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "helper", Ref: "org/repo/path@v1"},
	}
	lock := manifest.NewLockFile() // empty lock — not synced
	fs := newTestFileWriter(".github/agents/helper.agent.md") // but file exists

	results := CheckAssets(entries, lock, fs)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if results[0].Status != CheckNotInLock {
		t.Errorf("status = %d, want CheckNotInLock", results[0].Status)
	}
}

func TestCheckAssets_RefMismatch(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "helper", Ref: "org/repo/path@v2.0"},
	}
	lock := manifest.NewLockFile()
	lock.Set("agents", "helper", "org/repo/path@v1.0", "sha1", ".github/agents/helper.agent.md", []byte("x"))

	fs := newTestFileWriter(".github/agents/helper.agent.md")

	results := CheckAssets(entries, lock, fs)

	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	r := results[0]
	if r.Status != CheckRefMismatch {
		t.Errorf("status = %d, want CheckRefMismatch", r.Status)
	}
	if r.LockRef != "org/repo/path@v1.0" {
		t.Errorf("LockRef = %q, want %q", r.LockRef, "org/repo/path@v1.0")
	}
	if r.ManifRef != "org/repo/path@v2.0" {
		t.Errorf("ManifRef = %q, want %q", r.ManifRef, "org/repo/path@v2.0")
	}
}

func TestCheckAssets_Empty(t *testing.T) {
	t.Parallel()

	results := CheckAssets([]manifest.Entry{}, manifest.NewLockFile(), newTestFileWriter())

	if results == nil {
		t.Error("expected non-nil slice for empty entries")
	}
	if len(results) != 0 {
		t.Errorf("got %d results, want 0", len(results))
	}
}

func TestCheckAssets_MixedStatuses(t *testing.T) {
	t.Parallel()

	entries := []manifest.Entry{
		{Type: "agents", Name: "ok-agent", Ref: "org/repo/path@v1"},
		{Type: "agents", Name: "never-synced", Ref: "org/repo/path@v1"},
		{Type: "instructions", Name: "file-missing", Ref: "org/repo/inst@v1"},
		{Type: "prompts", Name: "not-in-lock", Ref: "org/repo/prompt@v1"},
		{Type: "agents", Name: "ref-mismatch", Ref: "org/repo/path@v2"},
	}

	lock := manifest.NewLockFile()
	lock.Set("agents", "ok-agent", "org/repo/path@v1", "sha", ".github/agents/ok-agent.agent.md", []byte("x"))
	lock.Set("instructions", "file-missing", "org/repo/inst@v1", "sha", ".github/instructions/file-missing.instructions.md", []byte("x"))
	lock.Set("agents", "ref-mismatch", "org/repo/path@v1", "sha", ".github/agents/ref-mismatch.agent.md", []byte("x"))

	fs := newTestFileWriter(
		".github/agents/ok-agent.agent.md",
		// never-synced: not on disk
		// file-missing: in lock but NOT on disk
		".github/prompts/not-in-lock.prompt.md", // on disk but NOT in lock
		".github/agents/ref-mismatch.agent.md",
	)

	results := CheckAssets(entries, lock, fs)

	if len(results) != 5 {
		t.Fatalf("got %d results, want 5", len(results))
	}

	byName := make(map[string]CheckResult, len(results))
	for _, r := range results {
		byName[r.Name] = r
	}

	wantStatuses := map[string]CheckStatus{
		"ok-agent":     CheckOK,
		"never-synced": CheckNeverSynced,
		"file-missing": CheckFileMissing,
		"not-in-lock":  CheckNotInLock,
		"ref-mismatch": CheckRefMismatch,
	}

	for name, want := range wantStatuses {
		r, ok := byName[name]
		if !ok {
			t.Errorf("missing result for %q", name)
			continue
		}
		if r.Status != want {
			t.Errorf("%q: status = %d, want %d", name, r.Status, want)
		}
	}
}
