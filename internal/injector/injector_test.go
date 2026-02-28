package injector

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cbout22/copilot-sync/internal/config"
	"github.com/cbout22/copilot-sync/internal/manifest"
	"github.com/cbout22/copilot-sync/internal/resolver"
)

// fakeSource implements resolver.SourceRepository for testing.
type fakeSource struct {
	files  map[string][]byte           // AssetRef.Path → content
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

// --- Phase 1: Injector with fakeSource (real filesystem via t.TempDir) ---

func newTestInjector(t *testing.T, src *fakeSource) (*Injector, string) {
	t.Helper()
	dir := t.TempDir()
	lock := manifest.NewLockFile()
	inj := New(src, lock, dir)
	return inj, dir
}

func TestInject_SingleFile_Success(t *testing.T) {
	t.Parallel()
	const rawRef = "org/repo/path/agent.md@v1"
	src := &fakeSource{
		files: map[string][]byte{"path/agent.md": []byte("agent content")},
		sha:   "abc123",
	}
	inj, dir := newTestInjector(t, src)

	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err != nil {
		t.Fatalf("Inject() error: %v", result.Err)
	}

	targetPath := filepath.Join(dir, ".github", "agents", "test.agent.md")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("file not written: %v", err)
	}
	if string(data) != "agent content" {
		t.Errorf("content = %q, want %q", string(data), "agent content")
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
	inj, _ := newTestInjector(t, src)

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
	inj, _ := newTestInjector(t, src)

	result := inj.Inject(config.Agents, "test", rawRef)
	if result.Err == nil {
		t.Fatal("expected error from SHA resolution, got nil")
	}
	if !strings.Contains(result.Err.Error(), "resolving commit SHA") {
		t.Errorf("error = %q, want it to contain 'resolving commit SHA'", result.Err.Error())
	}
}
