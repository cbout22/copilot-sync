package config

import (
	"path/filepath"
	"testing"
)

func TestAssetTypeIsValid(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input AssetType
		want  bool
	}{
		{Instructions, true},
		{Agents, true},
		{Prompts, true},
		{Skills, true},
		{"unknown", false},
		{"", false},
		{"INSTRUCTIONS", false},
	}
	for _, tc := range cases {
		if got := tc.input.IsValid(); got != tc.want {
			t.Errorf("AssetType(%q).IsValid() = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestAssetTypeFileExtension(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input AssetType
		want  string
	}{
		{Instructions, ".instructions.md"},
		{Agents, ".agent.md"},
		{Prompts, ".prompt.md"},
		{Skills, ""},
	}
	for _, tc := range cases {
		if got := tc.input.FileExtension(); got != tc.want {
			t.Errorf("AssetType(%q).FileExtension() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAssetTypeTargetDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input AssetType
		want  string
	}{
		{Instructions, filepath.Join(".github", "instructions")},
		{Agents, filepath.Join(".github", "agents")},
		{Prompts, filepath.Join(".github", "prompts")},
		{Skills, filepath.Join(".github", "skills")},
	}
	for _, tc := range cases {
		if got := tc.input.TargetDir(); got != tc.want {
			t.Errorf("AssetType(%q).TargetDir() = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestAssetTypeTargetPath_FileTypes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		assetType AssetType
		name      string
		want      string
	}{
		{Instructions, "my-review", filepath.Join(".github", "instructions", "my-review.instructions.md")},
		{Agents, "helper", filepath.Join(".github", "agents", "helper.agent.md")},
		{Prompts, "deploy", filepath.Join(".github", "prompts", "deploy.prompt.md")},
	}
	for _, tc := range cases {
		if got := tc.assetType.TargetPath(tc.name); got != tc.want {
			t.Errorf("%s.TargetPath(%q) = %q, want %q", tc.assetType, tc.name, got, tc.want)
		}
	}
}

func TestAssetTypeTargetPath_Skills(t *testing.T) {
	t.Parallel()
	got := Skills.TargetPath("my-skill")
	want := filepath.Join(".github", "skills", "my-skill")
	if got != want {
		t.Errorf("Skills.TargetPath(\"my-skill\") = %q, want %q", got, want)
	}
}

func TestAssetTypeIsDirectory(t *testing.T) {
	t.Parallel()
	if !Skills.IsDirectory() {
		t.Error("Skills.IsDirectory() = false, want true")
	}
	for _, at := range []AssetType{Instructions, Agents, Prompts} {
		if at.IsDirectory() {
			t.Errorf("%s.IsDirectory() = true, want false", at)
		}
	}
}

func TestParseRef_Valid_Simple(t *testing.T) {
	t.Parallel()
	ref, err := ParseRef("org/repo/path/to/file.md@v1.2.3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Org != "org" {
		t.Errorf("Org = %q, want %q", ref.Org, "org")
	}
	if ref.Repo != "repo" {
		t.Errorf("Repo = %q, want %q", ref.Repo, "repo")
	}
	if ref.Path != "path/to/file.md" {
		t.Errorf("Path = %q, want %q", ref.Path, "path/to/file.md")
	}
	if ref.Ref != "v1.2.3" {
		t.Errorf("Ref = %q, want %q", ref.Ref, "v1.2.3")
	}
}

func TestParseRef_Valid_NestedPath(t *testing.T) {
	t.Parallel()
	ref, err := ParseRef("myorg/myrepo/a/b/c/d@main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Path != "a/b/c/d" {
		t.Errorf("Path = %q, want %q", ref.Path, "a/b/c/d")
	}
}

func TestParseRef_ErrorCases(t *testing.T) {
	t.Parallel()
	cases := []string{
		"no-at-sign",
		"org/repo/path@",
		"@v1",
		"org/repo@v1",
		"/repo/path@v1",
		"org//path@v1",
		"org/repo/@v1",
	}
	for _, raw := range cases {
		_, err := ParseRef(raw)
		if err == nil {
			t.Errorf("ParseRef(%q) expected error, got nil", raw)
		}
	}
}

func TestAssetRefRaw_Roundtrip(t *testing.T) {
	t.Parallel()
	raw := "acme/configs/prompts/deploy.prompt.md@abc123"
	ref, err := ParseRef(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := ref.Raw(); got != raw {
		t.Errorf("Raw() roundtrip failed: got %q, want %q", got, raw)
	}
}

func TestAssetRefRaw_Deterministic(t *testing.T) {
	t.Parallel()
	r := AssetRef{Org: "o", Repo: "r", Path: "p/f.md", Ref: "v1"}
	want := "o/r/p/f.md@v1"
	for i := 0; i < 100; i++ {
		if got := r.Raw(); got != want {
			t.Fatalf("iteration %d: Raw() = %q, want %q", i, got, want)
		}
	}
}

func TestAssetRefRepoFullName(t *testing.T) {
	t.Parallel()
	r := AssetRef{Org: "myorg", Repo: "myrepo"}
	want := "myorg/myrepo"
	if got := r.RepoFullName(); got != want {
		t.Errorf("RepoFullName() = %q, want %q", got, want)
	}
}
