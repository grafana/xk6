package sync

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync/atomic"
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

func TestFindK6Require_V1(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("go.k6.io/k6", "v0.55.0")},
		},
	}

	path, version, found := findK6Require(mf)
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

func TestFindK6Require_V2(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("go.k6.io/k6/v2", "v2.0.0")},
		},
	}

	path, version, found := findK6Require(mf)
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

func TestFindK6Require_V3(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("go.k6.io/k6/v3", "v3.1.0")},
		},
	}

	path, version, found := findK6Require(mf)
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

func TestFindK6Require_NotFound(t *testing.T) {
	t.Parallel()

	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("github.com/foo/bar", "v1.2.3")},
			{Mod: mod("go.k6.io/xk6", "v1.0.0")}, // not k6 itself
		},
	}

	_, _, found := findK6Require(mf)
	if found {
		t.Fatal("did not expect to find k6 module")
	}
}

func TestResolveK6Module_ExplicitV1Version(t *testing.T) {
	t.Parallel()

	opts := &Options{K6Version: "v1.0.0"}
	mf := &modfile.File{}

	path, version, err := resolveK6Module(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6BaseModule {
		t.Errorf("expected path %s, got %s", k6BaseModule, path)
	}

	if version != "v1.0.0" {
		t.Errorf("expected version v1.0.0, got %s", version)
	}
}

func TestResolveK6Module_ExplicitV2Version(t *testing.T) {
	t.Parallel()

	opts := &Options{K6Version: "v2.0.0"}
	mf := &modfile.File{}

	path, version, err := resolveK6Module(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	const k6v2 = k6BaseModule + "/v2"
	if path != k6v2 {
		t.Errorf("expected path %s, got %s", k6v2, path)
	}

	if version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", version)
	}
}

func TestResolveK6Module_ExplicitV2SHA(t *testing.T) {
	const sha = "def9876543210"
	const pseudo = "v1.7.1-0.20260401000000-def9876543210"

	k6v2 := k6BaseModule + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6BaseModule, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		case fmt.Sprintf("/%s/@v/%s.mod", k6BaseModule, pseudo):
			_, _ = fmt.Fprint(w, modFileBody(k6v2))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	opts := &Options{K6Version: sha}
	mf := &modfile.File{}

	path, version, err := resolveK6Module(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6v2 {
		t.Errorf("expected path %s, got %s", k6v2, path)
	}

	if version != sha {
		t.Errorf("expected version %s, got %s", sha, version)
	}
}

func TestResolveK6Module_V1FromGoMod(t *testing.T) {
	t.Parallel()

	opts := &Options{}
	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("go.k6.io/k6", "v0.55.0")},
		},
	}

	path, version, err := resolveK6Module(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != "go.k6.io/k6" {
		t.Errorf("expected path go.k6.io/k6, got %s", path)
	}

	if version != "v0.55.0" {
		t.Errorf("expected version v0.55.0, got %s", version)
	}
}

func TestResolveK6Module_V2FromGoMod(t *testing.T) {
	t.Parallel()

	opts := &Options{}
	mf := &modfile.File{
		Require: []*modfile.Require{
			{Mod: mod("go.k6.io/k6/v2", "v2.0.0")},
		},
	}

	path, version, err := resolveK6Module(t.Context(), opts, mf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != "go.k6.io/k6/v2" {
		t.Errorf("expected path go.k6.io/k6/v2, got %s", path)
	}

	if version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", version)
	}
}

func TestGetOverallLatestK6Version_OnlyV1(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/go.k6.io/k6/@latest" {
			_, _ = fmt.Fprintf(w, `{"version":"v0.55.0"}`)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, version, err := getOverallLatestK6Version(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "go.k6.io/k6" {
		t.Errorf("expected path go.k6.io/k6, got %s", path)
	}
	if version != "v0.55.0" {
		t.Errorf("expected version v0.55.0, got %s", version)
	}
}

func TestGetOverallLatestK6Version_V2(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go.k6.io/k6/@latest":
			_, _ = fmt.Fprintf(w, `{"version":"v0.55.0"}`)
		case "/go.k6.io/k6/v2/@latest":
			_, _ = fmt.Fprintf(w, `{"version":"v2.0.0"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, version, err := getOverallLatestK6Version(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "go.k6.io/k6/v2" {
		t.Errorf("expected path go.k6.io/k6/v2, got %s", path)
	}
	if version != "v2.0.0" {
		t.Errorf("expected version v2.0.0, got %s", version)
	}
}

func TestGetOverallLatestK6Version_MultipleVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go.k6.io/k6/@latest":
			_, _ = fmt.Fprintf(w, `{"version":"v0.55.0"}`)
		case "/go.k6.io/k6/v2/@latest":
			_, _ = fmt.Fprintf(w, `{"version":"v2.1.0"}`)
		case "/go.k6.io/k6/v3/@latest":
			_, _ = fmt.Fprintf(w, `{"version":"v3.0.0"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, version, err := getOverallLatestK6Version(t.Context())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "go.k6.io/k6/v3" {
		t.Errorf("expected path go.k6.io/k6/v3, got %s", path)
	}
	if version != "v3.0.0" {
		t.Errorf("expected version v3.0.0, got %s", version)
	}
}

// modInfo returns a JSON .info response body.
func modInfo(version string) string {
	return fmt.Sprintf(`{"Version":%q,"Time":"2026-04-01T00:00:00Z"}`, version)
}

// modFile returns a minimal go.mod body declaring the given module path.
func modFileBody(modulePath string) string {
	return fmt.Sprintf("module %s\n\ngo 1.21\n", modulePath)
}

func TestProbeK6ModuleForVersion_V1SHA(t *testing.T) {
	const sha = "abc1234567890"
	const pseudo = "v0.55.1-0.20260401000000-abc1234567890"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6BaseModule, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		case fmt.Sprintf("/%s/@v/%s.mod", k6BaseModule, pseudo):
			_, _ = fmt.Fprint(w, modFileBody(k6BaseModule))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, err := probeK6ModuleForVersion(t.Context(), sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6BaseModule {
		t.Errorf("expected %s, got %s", k6BaseModule, path)
	}
}

func TestProbeK6ModuleForVersion_V2SHA(t *testing.T) {
	// The proxy serves the SHA under the base module but the go.mod inside
	// declares go.k6.io/k6/v2 — this is the authoritative path.
	const sha = "def9876543210"
	const pseudo = "v1.7.1-0.20260401000000-def9876543210"

	k6v2 := k6BaseModule + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6BaseModule, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		case fmt.Sprintf("/%s/@v/%s.mod", k6BaseModule, pseudo):
			_, _ = fmt.Fprint(w, modFileBody(k6v2))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, err := probeK6ModuleForVersion(t.Context(), sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6v2 {
		t.Errorf("expected %s, got %s", k6v2, path)
	}
}

func TestProbeK6ModuleForVersion_V2SHABaseNotFound(t *testing.T) {
	// The base module returns 404 for the SHA; only v2 has it.
	const sha = "aabbccddeeff"
	const pseudo = "v2.0.1-0.20260401000000-aabbccddeeff"

	k6v2 := k6BaseModule + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6v2, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		case fmt.Sprintf("/%s/@v/%s.mod", k6v2, pseudo):
			_, _ = fmt.Fprint(w, modFileBody(k6v2))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, err := probeK6ModuleForVersion(t.Context(), sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6v2 {
		t.Errorf("expected %s, got %s", k6v2, path)
	}
}

func TestGoProxyGet_RetriesOn5xx(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := attempts.Add(1)
		if n < 3 {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}
		_, _ = fmt.Fprint(w, `{"version":"v1.0.0"}`)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	resp, err := goProxyGet(t.Context(), "/go.k6.io/k6/@latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestGoProxyGet_ExhaustsRetries(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := goProxyGet(t.Context(), "/go.k6.io/k6/@latest")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	if attempts.Load() != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts.Load())
	}
}

func TestGoProxyGet_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	resp, err := goProxyGet(t.Context(), "/go.k6.io/k6/v99/@latest")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}

	if attempts.Load() != 1 {
		t.Errorf("expected exactly 1 attempt for 4xx, got %d", attempts.Load())
	}
}

func TestGetLatestVersion_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := getLatestVersion(t.Context(), "go.k6.io/k6/v2")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	if !errors.Is(err, errHTTP) {
		t.Errorf("expected errHTTP, got %v", err)
	}
}

func TestGetVersionInfo_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := getVersionInfo(t.Context(), "go.k6.io/k6/v2", "abc1234")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	if !errors.Is(err, errHTTP) {
		t.Errorf("expected errHTTP, got %v", err)
	}
}

func TestGetModule_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := getModule(t.Context(), "go.k6.io/k6/v2", "v2.0.0")
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}

	if !errors.Is(err, errHTTP) {
		t.Errorf("expected errHTTP, got %v", err)
	}
}

func mod(path, version string) module.Version {
	return module.Version{
		Path:    path,
		Version: version,
	}
}
