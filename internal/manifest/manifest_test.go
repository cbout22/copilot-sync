package manifest

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// --- helpers ---

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func tempPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

func readBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// --- New ---

func TestNew_EmptyInitializedMaps(t *testing.T) {
	t.Parallel()
	m := New()
	if m.Instructions == nil {
		t.Error("Instructions is nil")
	}
	if m.Agents == nil {
		t.Error("Agents is nil")
	}
	if m.Prompts == nil {
		t.Error("Prompts is nil")
	}
	if m.Skills == nil {
		t.Error("Skills is nil")
	}
	if len(m.Instructions) != 0 || len(m.Agents) != 0 || len(m.Prompts) != 0 || len(m.Skills) != 0 {
		t.Error("maps should be empty")
	}
}

// --- Set / Section ---

func TestManifest_Set_ValidTypes(t *testing.T) {
	t.Parallel()
	cases := []struct{ typ, name, ref string }{
		{"instructions", "my-inst", "org/repo/f.md@v1"},
		{"agents", "my-agent", "org/repo/a.md@v2"},
		{"prompts", "my-prompt", "org/repo/p.md@v3"},
		{"skills", "my-skill", "org/repo/s@main"},
	}
	for _, tc := range cases {
		m := New()
		if err := m.Set(tc.typ, tc.name, tc.ref); err != nil {
			t.Fatalf("Set(%q, %q, %q) error: %v", tc.typ, tc.name, tc.ref, err)
		}
		sec, err := m.Section(tc.typ)
		if err != nil {
			t.Fatalf("Section(%q) error: %v", tc.typ, err)
		}
		if sec[tc.name] != tc.ref {
			t.Errorf("Section(%q)[%q] = %q, want %q", tc.typ, tc.name, sec[tc.name], tc.ref)
		}
	}
}

func TestManifest_Set_UnknownType(t *testing.T) {
	t.Parallel()
	m := New()
	if err := m.Set("badtype", "foo", "bar"); err == nil {
		t.Error("expected error for unknown type, got nil")
	}
}

// --- Remove ---

func TestManifest_Remove_Existing(t *testing.T) {
	t.Parallel()
	m := New()
	_ = m.Set("agents", "a", "ref")
	removed, err := m.Remove("agents", "a")
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Error("expected removed=true")
	}
	sec, _ := m.Section("agents")
	if _, ok := sec["a"]; ok {
		t.Error("key still present after Remove")
	}
}

func TestManifest_Remove_Nonexistent(t *testing.T) {
	t.Parallel()
	m := New()
	removed, err := m.Remove("agents", "ghost")
	if err != nil {
		t.Fatal(err)
	}
	if removed {
		t.Error("expected removed=false for non-existent key")
	}
}

func TestManifest_Remove_UnknownType(t *testing.T) {
	t.Parallel()
	_, err := New().Remove("badtype", "x")
	if err == nil {
		t.Error("expected error for unknown type")
	}
}

// --- AllEntries ---

func TestManifest_AllEntries_Coverage(t *testing.T) {
	t.Parallel()
	m := New()
	_ = m.Set("instructions", "i", "ri")
	_ = m.Set("agents", "a", "ra")
	_ = m.Set("prompts", "p", "rp")
	_ = m.Set("skills", "s", "rs")

	entries := m.AllEntries()
	if len(entries) != 4 {
		t.Fatalf("AllEntries() returned %d entries, want 4", len(entries))
	}

	seen := make(map[string]bool)
	for _, e := range entries {
		seen[e.Type+"/"+e.Name] = true
	}
	for _, key := range []string{"instructions/i", "agents/a", "prompts/p", "skills/s"} {
		if !seen[key] {
			t.Errorf("AllEntries() missing %q", key)
		}
	}
}

// --- Load ---

func TestLoad_MissingFile_ReturnsEmptyManifest(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "does-not-exist.toml")
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.AllEntries()) != 0 {
		t.Error("expected empty manifest for missing file")
	}
}

func TestLoad_ValidTOML(t *testing.T) {
	t.Parallel()
	content := `[instructions]
  reviews = "org/repo/f.md@v1"

[agents]
  helper = "org/repo/a.md@v2"
`
	path := writeTempFile(t, "copilot.toml", content)
	m, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m.Instructions["reviews"] != "org/repo/f.md@v1" {
		t.Errorf("Instructions[reviews] = %q", m.Instructions["reviews"])
	}
	if m.Agents["helper"] != "org/repo/a.md@v2" {
		t.Errorf("Agents[helper] = %q", m.Agents["helper"])
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	t.Parallel()
	path := writeTempFile(t, "bad.toml", "[[[[invalid")
	_, err := Load(path)
	if err == nil {
		t.Error("expected parse error for invalid TOML")
	}
}

// --- Save + Roundtrip ---

func TestSave_Roundtrip(t *testing.T) {
	t.Parallel()
	m1 := New()
	_ = m1.Set("instructions", "r", "org/repo/f.md@main")
	_ = m1.Set("agents", "helper", "org/repo/a.md@v1")

	path := tempPath(t, "copilot.toml")
	if err := m1.Save(path); err != nil {
		t.Fatal(err)
	}

	m2, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if m2.Instructions["r"] != "org/repo/f.md@main" {
		t.Errorf("Instructions[r] = %q after roundtrip", m2.Instructions["r"])
	}
	if m2.Agents["helper"] != "org/repo/a.md@v1" {
		t.Errorf("Agents[helper] = %q after roundtrip", m2.Agents["helper"])
	}
	if len(m2.Prompts) != 0 {
		t.Errorf("Prompts should be empty, got %v", m2.Prompts)
	}
	if len(m2.Skills) != 0 {
		t.Errorf("Skills should be empty, got %v", m2.Skills)
	}
}

// --- Determinism ---

func TestSave_DeterministicTOML(t *testing.T) {
	t.Parallel()
	m := New()
	// Add keys in reverse-alphabetical order to prove sorting happens
	_ = m.Set("instructions", "z-review", "org/repo/z.md@v1")
	_ = m.Set("instructions", "a-review", "org/repo/a.md@v1")
	_ = m.Set("agents", "b-agent", "org/repo/b.md@v1")

	path1 := tempPath(t, "first.toml")
	path2 := tempPath(t, "second.toml")
	if err := m.Save(path1); err != nil {
		t.Fatal(err)
	}
	if err := m.Save(path2); err != nil {
		t.Fatal(err)
	}

	b1 := readBytes(t, path1)
	b2 := readBytes(t, path2)
	if !bytes.Equal(b1, b2) {
		t.Errorf("non-deterministic TOML output:\nfirst:\n%s\nsecond:\n%s", b1, b2)
	}
}

func TestSave_DeterministicTOML_MultipleRuns(t *testing.T) {
	t.Parallel()
	// Run save 20 times and ensure all outputs are byte-identical
	m := New()
	_ = m.Set("instructions", "z", "org/repo/z.md@v1")
	_ = m.Set("instructions", "m", "org/repo/m.md@v1")
	_ = m.Set("instructions", "a", "org/repo/a.md@v1")
	_ = m.Set("agents", "x", "org/repo/x.md@v1")
	_ = m.Set("prompts", "q", "org/repo/q.md@v1")

	var reference []byte
	for i := 0; i < 20; i++ {
		path := tempPath(t, "run.toml")
		if err := m.Save(path); err != nil {
			t.Fatal(err)
		}
		data := readBytes(t, path)
		if i == 0 {
			reference = data
		} else if !bytes.Equal(data, reference) {
			t.Fatalf("iteration %d: TOML output differs:\ngot:\n%s\nwant:\n%s", i, data, reference)
		}
	}
}

func TestSave_TOML_KeysSorted(t *testing.T) {
	t.Parallel()
	m := New()
	_ = m.Set("instructions", "c-inst", "org/repo/c.md@v1")
	_ = m.Set("instructions", "a-inst", "org/repo/a.md@v1")
	_ = m.Set("instructions", "b-inst", "org/repo/b.md@v1")

	path := tempPath(t, "sorted.toml")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got := string(readBytes(t, path))

	// Keys must appear in sorted order within the section
	posA := bytes.Index([]byte(got), []byte("a-inst"))
	posB := bytes.Index([]byte(got), []byte("b-inst"))
	posC := bytes.Index([]byte(got), []byte("c-inst"))
	if !(posA < posB && posB < posC) {
		t.Errorf("TOML keys not sorted: a=%d b=%d c=%d\noutput:\n%s", posA, posB, posC, got)
	}
}

func TestSave_TOML_SectionsFollowStructOrder(t *testing.T) {
	t.Parallel()
	m := New()
	// Add entries in reverse struct-field order
	_ = m.Set("skills", "s", "org/repo/s@v1")
	_ = m.Set("prompts", "p", "org/repo/p@v1")
	_ = m.Set("agents", "a", "org/repo/a@v1")
	_ = m.Set("instructions", "i", "org/repo/i@v1")

	path := tempPath(t, "sections.toml")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got := string(readBytes(t, path))

	// Struct field declaration order: Instructions, Agents, Prompts, Skills
	sections := []string{"[instructions]", "[agents]", "[prompts]", "[skills]"}
	positions := make([]int, len(sections))
	for i, sec := range sections {
		pos := bytes.Index([]byte(got), []byte(sec))
		if pos < 0 {
			t.Fatalf("section %q not found in output:\n%s", sec, got)
		}
		positions[i] = pos
	}
	if !sort.IntsAreSorted(positions) {
		t.Errorf("sections not in struct declaration order: positions=%v\noutput:\n%s", positions, got)
	}
}

func TestSave_TOML_EmptySectionsOmitted(t *testing.T) {
	t.Parallel()
	m := New()
	_ = m.Set("instructions", "r", "org/repo/r@v1")
	// agents, prompts, skills are empty

	path := tempPath(t, "omit.toml")
	if err := m.Save(path); err != nil {
		t.Fatal(err)
	}
	got := string(readBytes(t, path))

	for _, sec := range []string{"[agents]", "[prompts]", "[skills]"} {
		if bytes.Contains([]byte(got), []byte(sec)) {
			t.Errorf("empty section %q should be omitted, but found in output:\n%s", sec, got)
		}
	}
}
