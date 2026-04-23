package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveK6Repo_CustomV2SHA(t *testing.T) {
	const base = "github.com/myfork/k6"
	const sha = "f520efb45f42"
	const pseudo = "v2.0.1-0.20260401000000-f520efb45f42"

	basev2 := base + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", basev2, sha):
			_, _ = fmt.Fprintf(w, `{"Version":%q,"Time":"2026-04-01T00:00:00Z"}`, pseudo)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	opts := &buildOptions{
		extensions:   new(modules),
		replacements: &modules{replace: true},
		k6repo:       base,
		k6version:    sha,
	}

	resolveK6Repo(t.Context(), opts)

	if opts.k6repo != basev2 {
		t.Errorf("expected k6repo %s, got %s", basev2, opts.k6repo)
	}
}

func TestResolveK6Repo_CustomV1SHA(t *testing.T) {
	const base = "github.com/myfork/k6"
	const sha = "aabbccddeeff"
	const pseudo = "v0.55.0-0.20260401000000-aabbccddeeff"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@v/%s.info", base, sha):
			_, _ = fmt.Fprintf(w, `{"Version":%q,"Time":"2026-04-01T00:00:00Z"}`, pseudo)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	opts := &buildOptions{
		extensions:   new(modules),
		replacements: &modules{replace: true},
		k6repo:       base,
		k6version:    sha,
	}

	resolveK6Repo(t.Context(), opts)

	if opts.k6repo != base {
		t.Errorf("expected k6repo %s (unchanged), got %s", base, opts.k6repo)
	}
}

func TestResolveK6Repo_ExplicitV2SuffixSkipsProbe(t *testing.T) {
	// When the repo already has /v2, resolveK6Repo should not probe.
	const repo = "github.com/myfork/k6/v2"
	const sha = "deadbeefcafe"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected proxy request: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	opts := &buildOptions{
		extensions:   new(modules),
		replacements: &modules{replace: true},
		k6repo:       repo,
		k6version:    sha,
	}

	resolveK6Repo(t.Context(), opts)

	if opts.k6repo != repo {
		t.Errorf("expected k6repo %s (unchanged), got %s", repo, opts.k6repo)
	}
}

func TestResolveK6Repo_CustomLatest(t *testing.T) {
	const base = "github.com/myfork/k6"
	basev2 := base + "/v2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case fmt.Sprintf("/%s/@latest", base):
			_, _ = fmt.Fprint(w, `{"version":"v0.55.0"}`)
		case fmt.Sprintf("/%s/@latest", basev2):
			_, _ = fmt.Fprint(w, `{"version":"v2.1.0"}`)
		case fmt.Sprintf("/%s/v3/@latest", base):
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	t.Setenv("GOPROXY", srv.URL)

	opts := &buildOptions{
		extensions:   new(modules),
		replacements: &modules{replace: true},
		k6repo:       base,
		k6version:    defaultK6Version,
	}

	resolveK6Repo(t.Context(), opts)

	if opts.k6repo != basev2 {
		t.Errorf("expected k6repo %s, got %s", basev2, opts.k6repo)
	}

	if opts.k6version != "v2.1.0" {
		t.Errorf("expected k6version v2.1.0, got %s", opts.k6version)
	}
}
