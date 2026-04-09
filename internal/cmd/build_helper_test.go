package cmd

import (
	"testing"
)

func TestFindK6ModVersion_V1(t *testing.T) {
	t.Parallel()

	modVersions := map[string]string{
		"go.k6.io/k6":               "v0.55.0",
		"github.com/some/extension": "v1.2.3",
	}

	path, version, found := findK6ModVersion(modVersions)
	if !found {
		t.Fatal("expected to find k6 module")
	}

	if path != "go.k6.io/k6" {
		t.Errorf("expected path go.k6.io/k6, got %s", path)
	}

	if version != "v0.55.0" {
		t.Errorf("expected version v0.55.0, got %s", version)
	}
}

func TestFindK6ModVersion_V2(t *testing.T) {
	t.Parallel()

	modVersions := map[string]string{
		"go.k6.io/k6/v2":            "v2.0.0",
		"github.com/some/extension": "v1.2.3",
	}

	path, version, found := findK6ModVersion(modVersions)
	if !found {
		t.Fatal("expected to find k6/v2 module")
	}

	if path != "go.k6.io/k6/v2" {
		t.Errorf("expected path go.k6.io/k6/v2, got %s", path)
	}

	if version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", version)
	}
}

func TestFindK6ModVersion_V3(t *testing.T) {
	t.Parallel()

	modVersions := map[string]string{
		"go.k6.io/k6/v3": "v3.1.0",
	}

	path, version, found := findK6ModVersion(modVersions)
	if !found {
		t.Fatal("expected to find k6/v3 module")
	}

	if path != "go.k6.io/k6/v3" {
		t.Errorf("expected path go.k6.io/k6/v3, got %s", path)
	}

	if version != "v3.1.0" {
		t.Errorf("expected version v3.1.0, got %s", version)
	}
}

func TestFindK6ModVersion_NotFound(t *testing.T) {
	t.Parallel()

	modVersions := map[string]string{
		"github.com/some/extension": "v1.2.3",
		"go.k6.io/xk6":              "v1.0.0", // not k6 itself
	}

	_, _, found := findK6ModVersion(modVersions)
	if found {
		t.Fatal("did not expect to find k6 module")
	}
}

func TestFindK6ModVersion_Empty(t *testing.T) {
	t.Parallel()

	modVersions := map[string]string{}

	_, _, found := findK6ModVersion(modVersions)
	if found {
		t.Fatal("did not expect to find k6 module in empty map")
	}
}
