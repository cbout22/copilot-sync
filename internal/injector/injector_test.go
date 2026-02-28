package injector

import (
	"bytes"
	"testing"
)

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
