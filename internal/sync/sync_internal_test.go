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

func TestResolveK6Module_ExplicitV1SHA(t *testing.T) {
	// A SHA from the v1 module: base .info succeeds → return baseModule directly.
	const sha = "def9876543210"
	const pseudo = "v0.55.1-0.20260401000000-def9876543210"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6BaseModule, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
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

	if path != k6BaseModule {
		t.Errorf("expected path %s, got %s", k6BaseModule, path)
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

func TestProbeK6ModuleForVersion_V1SHA(t *testing.T) {
	const sha = "abc1234567890"
	const pseudo = "v0.55.1-0.20260401000000-abc1234567890"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6BaseModule, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		default:
			// No .mod request should occur: base .info success → return baseModule directly.
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

func TestProbeK6ModuleForVersion_V2SHABaseNotFound(t *testing.T) {
	// The base module returns 404 for the SHA; v2 has it.
	// Short-circuit: once .info succeeds for a versioned path (/v2), we return
	// that path directly without fetching .mod.  The mock fails the test if
	// .mod is requested to prove the short-circuit is active.
	const sha = "aabbccddeeff"
	const pseudo = "v2.0.1-0.20260401000000-aabbccddeeff"

	k6v2 := k6BaseModule + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6v2, sha):
			_, _ = fmt.Fprint(w, modInfo(pseudo))
		default:
			// base .info and anything else (including v2 .mod) should be 404.
			// Using 404 (not 500) ensures the retry logic is not triggered for
			// the base module probe; 500 would cause 3 retries and a 3s delay.
			//
			// .mod for v2 should NOT be fetched due to the short-circuit, so any
			// test that receives a .mod request will see a 404 and surface the bug
			// as an unexpected HTTP error in the caller.
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

func TestProbeModuleVersionForBase_LokiV3RealProxy(t *testing.T) {
	t.Parallel()

	// Real-world case: the base module (github.com/grafana/loki) returns 404 for
	// SHA ac49d8321e968014b983053a3f3db3fcdba36f34; the fallback loop finds it at v3.
	const (
		base = "github.com/grafana/loki"
		sha  = "ac49d8321e968014b983053a3f3db3fcdba36f34"
	)

	path, err := probeModuleVersionForBase(t.Context(), base, sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if want := base + "/v3"; path != want {
		t.Errorf("expected %s, got %s", want, path)
	}
}

func TestProbeK6ModuleForVersion_V3SHASkipsV2(t *testing.T) {
	// v2 exists (/@latest returns a version) but does not contain the SHA;
	// the SHA is in v3.  The loop must continue past v2's 404 to find v3.
	//
	// Request sequence:
	//   base/.info  → 404
	//   v2/.info    → 404
	//   v2/@latest  → 200 (v2 exists → consecutiveAbsent stays 0, keep probing)
	//   v3/.info    → 200 (found; short-circuit, no .mod)
	const sha = "ccddee112233"
	const v3pseudo = "v3.1.0-0.20260401000000-ccddee112233"

	k6v2 := k6BaseModule + "/v2"
	k6v3 := k6BaseModule + "/v3"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@latest", k6v2):
			_, _ = fmt.Fprintf(w, `{"version":"v2.0.0"}`)
		case fmt.Sprintf("/%s/@v/%s.info", k6v3, sha):
			_, _ = fmt.Fprint(w, modInfo(v3pseudo))
		default:
			// base .info, v2 .info, and anything else → 404.
			// .mod for v3 should NOT be requested (short-circuit).
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, err := probeK6ModuleForVersion(t.Context(), sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6v3 {
		t.Errorf("expected %s, got %s", k6v3, path)
	}
}

func TestProbeK6ModuleForVersion_V3SHAAbsentV2(t *testing.T) {
	// v2 does not exist at the proxy (/@latest → 404) — the module went v1 → v3
	// directly, as loki does.  The loop must tolerate one absent major and keep
	// probing: consecutiveAbsent reaches 1 after v2, then v3 is found.
	//
	// Request sequence:
	//   base/.info  → 404
	//   v2/.info    → 404
	//   v2/@latest  → 404 (v2 absent → consecutiveAbsent=1, < maxConsecutiveAbsent=2)
	//   v3/.info    → 200 (found; short-circuit, no .mod)
	const sha = "aabb11223344"
	const v3pseudo = "v3.2.0-0.20260401000000-aabb11223344"

	k6v3 := k6BaseModule + "/v3"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", k6v3, sha):
			_, _ = fmt.Fprint(w, modInfo(v3pseudo))
		default:
			// base .info, v2 .info, v2 @latest → 404 (v2 does not exist).
			// .mod for v3 should NOT be requested (short-circuit).
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	path, err := probeK6ModuleForVersion(t.Context(), sha)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if path != k6v3 {
		t.Errorf("expected %s, got %s", k6v3, path)
	}
}

func TestProbeK6ModuleForVersion_UnknownSHA(t *testing.T) {
	// The SHA does not exist anywhere; the proxy returns 404 for every probe.
	// The loop must terminate after maxConsecutiveAbsent major versions are not found.
	const sha = "000000000000000000000000000000000000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := probeK6ModuleForVersion(t.Context(), sha)
	if err == nil {
		t.Fatal("expected error for unknown SHA, got nil")
	}
}

func TestGoProxyGet_RetriesOn5xx(t *testing.T) {
	var attempts atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts.Add(1)
		http.Error(w, "server error", http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	resp, err := goProxyGet(t.Context(), "/go.k6.io/k6/@latest")
	if resp != nil {
		_ = resp.Body.Close()
	}

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

func TestProbeVersionInfo_PlainNotFound(t *testing.T) {
	// A 404 response → errHTTP.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	_, err := probeVersionInfo(t.Context(), "go.k6.io/k6/v2", "abc1234")
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
