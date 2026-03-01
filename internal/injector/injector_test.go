package injector

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/resolver"
)

// ---- fakeSource ----

// fakeSource implements resolver.SourceRepository for testing.
type fakeSource struct {
	files  map[string][]byte // AssetRef.Path → content
	dirs   map[string][]resolver.GitHubTreeEntry
	sha    string
	failOn string // path that should return an error
}

func (f *fakeSource) DownloadFile(ref config.AssetRef) ([]byte, error) {
	if ref.Path == f.failOn {
		return nil, fmt.Errorf("simulated download failure for %s", ref.Path)
	}
	content, ok := f.files[ref.Path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", ref.Path)
	}
	return content, nil
}

func (f *fakeSource) ListDirectory(ref config.AssetRef) ([]resolver.GitHubTreeEntry, error) {
	entries, ok := f.dirs[ref.Path]
	if !ok {
		return nil, fmt.Errorf("directory not found: %s", ref.Path)
	}
	return entries, nil
}

func (f *fakeSource) ResolveSHA(ref config.AssetRef) (string, error) {
	if f.sha == "" {
		return "", fmt.Errorf("SHA resolution failed")
	}
	return f.sha, nil
}

// ---- memFileWriter ----

// memFileWriter implements FileWriter for testing — all operations in memory.
type memFileWriter struct {
	written map[string][]byte // path → data
	dirs    map[string]bool   // created directories
	removed []string          // paths removed
	failOn  string            // path that should return a write error
}

func newMemFileWriter() *memFileWriter {
	return &memFileWriter{
		written: make(map[string][]byte),
		dirs:    make(map[string]bool),
	}
}

func (m *memFileWriter) Write(path string, data []byte) error {
	if m.failOn != "" && path == m.failOn {
		return fmt.Errorf("simulated write failure for %s", path)
	}
	m.written[path] = append([]byte{}, data...) // defensive copy
	return nil
}

func (m *memFileWriter) MkdirAll(path string) error {
	m.dirs[path] = true
	return nil
}

func (m *memFileWriter) Remove(path string) error {
	m.removed = append(m.removed, path)
	delete(m.written, path)
	return nil
}

func (m *memFileWriter) Exists(path string) bool {
	_, ok := m.written[path]
	return ok
}

// ---- helpers ----

const rootDir = "/project"

func newTestInjector(src *fakeSource, mfw *memFileWriter) (*Injector, *manifest.LockFile) {
	lock := manifest.NewLockFile()
	inj := New(src, lock, rootDir, mfw)
	return inj, lock
}

// ---- computeDirectoryChecksum tests ----

func TestComputeDirectoryChecksum_Deterministic(t *testing.T) {
	t.Parallel()
	contents := map[string][]byte{
		"z_file.md": []byte("zzz content"),
		"m_file.md": []byte("mmm content"),
		"a_file.md": []byte("aaa content"),
	}
	reference := computeDirectoryChecksum(contents)
	for i := 0; i < 200; i++ {
		result := computeDirectoryChecksum(contents)
		if !bytes.Equal(result, reference) {
			t.Fatalf(
				"computeDirectoryChecksum is non-deterministic (iteration %d):\n"+
					"reference: %x\ngot:       %x",
				i, reference, result,
			)
		}
	}
}

func TestComputeDirectoryChecksum_Empty(t *testing.T) {
	t.Parallel()
	result := computeDirectoryChecksum(map[string][]byte{})
	if len(result) != 0 {
		t.Errorf("expected empty result for empty map, got %x", result)
	}
}

func TestComputeDirectoryChecksum_SingleFile(t *testing.T) {
	t.Parallel()
	content := []byte("single file content")
	result := computeDirectoryChecksum(map[string][]byte{"only.md": content})
	if !bytes.Equal(result, content) {
		t.Errorf("single file: expected raw content %q, got %q", content, result)
	}
}

func TestComputeDirectoryChecksum_SortedConcatenation(t *testing.T) {
	t.Parallel()
	result := computeDirectoryChecksum(map[string][]byte{
		"b.md": []byte("B"),
		"a.md": []byte("A"),
	})
	want := []byte("AB") // sorted: a.md content first, then b.md
	if !bytes.Equal(result, want) {
		t.Errorf("expected sorted concatenation %q, got %q", want, result)
	}
}

func TestComputeDirectoryChecksum_StableAcrossMultipleCalls(t *testing.T) {
	t.Parallel()
	contents := map[string][]byte{
		"e.md": []byte("E"),
		"a.md": []byte("A"),
		"c.md": []byte("C"),
		"b.md": []byte("B"),
		"d.md": []byte("D"),
	}
	first := computeDirectoryChecksum(contents)
	for i := 0; i < 50; i++ {
		if got := computeDirectoryChecksum(contents); !bytes.Equal(got, first) {
			t.Fatalf("non-deterministic on call %d: got %q, want %q", i, got, first)
		}
	}
	// Sorted keys: a b c d e → content "ABCDE"
	if !bytes.Equal(first, []byte("ABCDE")) {
		t.Errorf("expected ABCDE, got %q", first)
	}
}

// ---- Injector tests using fakeSource + memFileWriter ----

func TestInject_SingleFile_Success(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	src := &fakeSource{
		files: map[string][]byte{"path/agent.md": []byte("agent content")},
		sha:   "abc123",
	}
	mfw := newMemFileWriter()
	inj, _ := newTestInjector(src, mfw)

	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}
}

func TestInject_SingleFile_WritesCorrectPath(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/file.md@v1"
	wantContent := []byte("clean code instructions")
	src := &fakeSource{
		files: map[string][]byte{"path/file.md": wantContent},
		sha:   "sha1",
	}
	mfw := newMemFileWriter()
	inj, _ := newTestInjector(src, mfw)

	result := inj.Inject(config.Instructions, "clean-code", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}

	wantPath := filepath.Join(rootDir, ".github", "instructions", "clean-code.instructions.md")
	data, ok := mfw.written[wantPath]
	if !ok {
		t.Fatalf("expected file written at %q, got keys: %v", wantPath, keysOf(mfw.written))
	}
	if !bytes.Equal(data, wantContent) {
		t.Errorf("content = %q, want %q", data, wantContent)
	}
}

func TestInject_SingleFile_OverwriteExisting(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	targetPath := filepath.Join(rootDir, ".github", "agents", "test.agent.md")

	src := &fakeSource{
		files: map[string][]byte{"path/agent.md": []byte("new content")},
		sha:   "sha2",
	}
	mfw := newMemFileWriter()
	// Pre-populate so Exists() returns true
	mfw.written[targetPath] = []byte("old content")

	inj, _ := newTestInjector(src, mfw)
	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}

	// Remove should have been called for the old file
	if len(mfw.removed) == 0 {
		t.Error("expected Remove to be called for existing file")
	}
	found := false
	for _, p := range mfw.removed {
		if p == targetPath {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected %q to be in removed list, got: %v", targetPath, mfw.removed)
	}

	// New content should be written
	data := mfw.written[targetPath]
	if string(data) != "new content" {
		t.Errorf("content = %q, want %q", string(data), "new content")
	}
}

func TestInject_DownloadError(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	src := &fakeSource{
		files:  map[string][]byte{},
		sha:    "abc123",
		failOn: "path/agent.md",
	}
	mfw := newMemFileWriter()
	inj, _ := newTestInjector(src, mfw)

	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(result.Err.Error(), "simulated download failure") {
		t.Errorf("error = %q, want it to contain 'simulated download failure'", result.Err.Error())
	}
}

func TestInject_SHAResolutionFailure(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	src := &fakeSource{
		files: map[string][]byte{"path/agent.md": []byte("content")},
		sha:   "", // empty triggers error
	}
	mfw := newMemFileWriter()
	inj, _ := newTestInjector(src, mfw)

	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err == nil {
		t.Fatal("expected error from SHA resolution, got nil")
	}
	if !strings.Contains(result.Err.Error(), "resolving commit SHA") {
		t.Errorf("error = %q, want it to contain 'resolving commit SHA'", result.Err.Error())
	}
}

func TestInject_Directory_WritesAllFiles(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/skills/k8s@main"
	src := &fakeSource{
		files: map[string][]byte{
			"skills/k8s/deploy.md":   []byte("deploy"),
			"skills/k8s/rollback.md": []byte("rollback"),
			"skills/k8s/status.md":   []byte("status"),
		},
		dirs: map[string][]resolver.GitHubTreeEntry{
			"skills/k8s": {
				{Path: "skills/k8s/deploy.md", Type: "blob"},
				{Path: "skills/k8s/rollback.md", Type: "blob"},
				{Path: "skills/k8s/status.md", Type: "blob"},
			},
		},
		sha: "sha3",
	}
	mfw := newMemFileWriter()
	inj, _ := newTestInjector(src, mfw)

	result := inj.Inject(config.Skills, "k8s", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}

	wantFiles := []string{"deploy.md", "rollback.md", "status.md"}
	for _, name := range wantFiles {
		wantPath := filepath.Join(rootDir, ".github", "skills", "k8s", name)
		if _, ok := mfw.written[wantPath]; !ok {
			t.Errorf("expected file %q to be written, got keys: %v", wantPath, keysOf(mfw.written))
		}
	}
}

func TestInject_Directory_UpdatesLock(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/skills/k8s@main"
	src := &fakeSource{
		files: map[string][]byte{
			"skills/k8s/deploy.md": []byte("deploy"),
		},
		dirs: map[string][]resolver.GitHubTreeEntry{
			"skills/k8s": {
				{Path: "skills/k8s/deploy.md", Type: "blob"},
			},
		},
		sha: "sha-lock",
	}
	mfw := newMemFileWriter()
	inj, lock := newTestInjector(src, mfw)

	result := inj.Inject(config.Skills, "k8s", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}

	entry, ok := lock.Get("skills", "k8s")
	if !ok {
		t.Fatal("lock entry not set after skill inject")
	}
	if entry.ResolvedSHA != "sha-lock" {
		t.Errorf("lock SHA = %q, want %q", entry.ResolvedSHA, "sha-lock")
	}
	if entry.Checksum == "" {
		t.Error("lock Checksum is empty")
	}
}

func TestInject_WriteError_PropagatesError(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	targetPath := filepath.Join(rootDir, ".github", "agents", "test.agent.md")

	src := &fakeSource{
		files: map[string][]byte{"path/agent.md": []byte("content")},
		sha:   "sha4",
	}
	mfw := newMemFileWriter()
	mfw.failOn = targetPath

	inj, _ := newTestInjector(src, mfw)
	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err == nil {
		t.Fatal("expected write error to propagate, got nil")
	}
	if !strings.Contains(result.Err.Error(), "simulated write failure") {
		t.Errorf("error = %q, want it to contain 'simulated write failure'", result.Err.Error())
	}
}

// ---- helpers ----

func keysOf(m map[string][]byte) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
