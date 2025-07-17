package sync

import (
	"reflect"
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

	want := []*Change{
		{
			Module: "github.com/foo/bar",
			From:   "v1.2.3",
			To:     "v1.2.4",
		},
	}

	got := diffRequires(ext, k6)
	if reflect.DeepEqual(got, want) == false {
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

	want := []*Change{
		{
			Module: "github.com/foo/bar",

			From: "v1.2.3",
			To:   "v1.2.4",
		},
		{
			Module: "github.com/baz/qux",
			From:   "v2.0.0",
			To:     "v2.1.0",
		},
	}

	got := diffRequires(ext, k6)
	if !reflect.DeepEqual(got, want) {
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
