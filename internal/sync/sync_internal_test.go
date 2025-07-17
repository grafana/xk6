package sync

import (
	"testing"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func TestDiffRequires_NoDifferences(t *testing.T) {
	t.Parallel()

	ext := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("github.com/baz/qux", "v2.0.0")},
		},
	}

	k6 := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("github.com/baz/qux", "v2.0.0")},
		},
	}

	got := diffRequires(ext, k6)
	if len(got) != 0 {
		t.Errorf("expected no differences, got %v", got)
	}
}

func TestDiffRequires_OneDifference(t *testing.T) {
	t.Parallel()

	ext := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
		},
	}

	k6 := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.4")},
		},
	}

	want := []string{"github.com/foo/bar@v1.2.4"}

	got := diffRequires(ext, k6)
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDiffRequires_MultipleDifferences(t *testing.T) {
	t.Parallel()

	ext := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("github.com/baz/qux", "v2.0.0")},
		},
	}

	k6 := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.4")},
			{Mod: mod("github.com/baz/qux", "v2.1.0")},
		},
	}

	want := []string{
		"github.com/foo/bar@v1.2.4",
		"github.com/baz/qux@v2.1.0",
	}

	got := diffRequires(ext, k6)
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("expected %v, got %v", want, got)
	}
}

func TestDiffRequires_NoMatchingModules(t *testing.T) {
	t.Parallel()

	ext := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
		},
	}

	k6 := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/other/module", "v0.1.0")},
		},
	}

	got := diffRequires(ext, k6)
	if len(got) != 0 {
		t.Errorf("expected no differences, got %v", got)
	}
}

func TestFindRequire_Found(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("go.k6.io/k6", "v0.48.0")},
		},
	}

	version, found := findRequire(mf, "go.k6.io/k6")
	if !found {
		t.Fatalf("expected to find module")
	}

	if version != "v0.48.0" {
		t.Errorf("expected version v0.48.0, got %s", version)
	}
}

func TestFindRequire_NotFound(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
		},
	}

	version, found := findRequire(mf, "go.k6.io/k6")
	if found {
		t.Fatalf("did not expect to find module")
	}

	if version != "" {
		t.Errorf("expected empty version, got %s", version)
	}
}

func TestGetK6Version_WithExplicitVersion(t *testing.T) {
	t.Parallel()

	opts := &Options{K6Version: "v0.49.0"}
	mf := &modfile.File{}

	got, err := getK6Version(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "v0.49.0" {
		t.Errorf("expected v0.49.0, got %s", got)
	}
}

func TestGetK6Version_WithRequireVersion(t *testing.T) {
	t.Parallel()

	opts := &Options{}
	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod(k6Module, "v0.48.0")},
		},
	}

	got, err := getK6Version(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "v0.48.0" {
		t.Errorf("expected v0.48.0, got %s", got)
	}
}

func mod(path, version string) module.Version {
	return module.Version{
		Path:    path,
		Version: version,
	}
}
