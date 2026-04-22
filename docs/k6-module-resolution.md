# k6 module resolution

xk6 automatically resolves the correct k6 Go module path (`go.k6.io/k6`, `go.k6.io/k6/v2`, etc.) when `--k6-repo` is not explicitly set. This page explains why that is necessary and how the algorithm works.

## Background

Go requires a distinct module path for each major version beyond v1 (see [Go module compatibility](https://go.dev/blog/v2-go-modules)). When k6 moved from `go.k6.io/k6` to `go.k6.io/k6/v2`, any tool that hard-coded the base module path would fail with an error like:

```
go.mod:5: require go.k6.io/k6: version "v1.7.1-0.20260401084312-d2c34d930734" invalid:
  go.mod has post-v1 module path "go.k6.io/k6/v2" at revision d2c34d930734
```

Rather than require every user to add `--k6-repo go.k6.io/k6/v2`, xk6 determines the right module path automatically.

## Proxy protocol constraints

The resolution relies on the [Go module proxy protocol](https://go.dev/ref/mod#goproxy-protocol). Two endpoints are relevant:

| Endpoint | Accepts | Returns |
|---|---|---|
| `/$module/@v/$ref.info` | Any version reference — semver tag, commit SHA, branch name, pseudo-version | JSON `{"Version":"<canonical>","Time":"<RFC3339>"}` |
| `/$module/@v/$version.mod` | Canonical version only (semver tag or pseudo-version) | The `go.mod` file |

The `.info` endpoint is the only one that accepts raw SHAs and branch names. The `.mod` endpoint requires a canonical version. This two-step requirement shapes the SHA/branch resolution path below.

There is no proxy endpoint to enumerate all major-version module paths for a given module. Sequential probing (trying `/v2`, `/v3`, … until a 404 is returned) is the standard approach used by the Go toolchain itself.

## Entry point

`resolveK6Repo` in `internal/cmd/build_helper.go` is called at the start of every build (both `xk6 build` and `xk6 run`). If `--k6-repo` has been set explicitly, it returns immediately. Otherwise it delegates to one of two sub-algorithms based on whether `--k6-version` was provided.

## Algorithm A: no explicit version (`--k6-version latest`)

Implemented in `sync.ResolveK6ModuleForExtensions`.

### Step 1 — scan extension dependencies

For each extension passed via `--with`:

1. If the extension has a local replace path (e.g. when running `xk6 run` from inside an extension's own directory), read its `go.mod` from disk.
2. Otherwise, resolve the version to use:
   - If a version was pinned (e.g. `--with github.com/foo/bar@v1.2.3`), use it directly.
   - If no version was pinned, fetch `/$ext/@latest` from the proxy to get the latest version.
3. Fetch `/$ext/@v/$version.mod` from the proxy to get the extension's `go.mod`.
4. Scan the `require` directives for any entry whose module path is `go.k6.io/k6` or matches `go.k6.io/k6/v*`.
5. If found, return that module path and version immediately — no further probing needed.

The first extension that declares a k6 dependency wins. Extensions that cannot be read (network errors, no `go.mod`) are skipped with a debug log entry.

### Step 2 — fallback: probe for overall latest

If no extension declared k6 (or there are no extensions), `getOverallLatestK6Version` is called:

1. Fetch `go.k6.io/k6/@latest` — record version as `bestVersion`.
2. Fetch `go.k6.io/k6/v2/@latest`. If successful and higher than `bestVersion`, update both.
3. Continue with `/v3`, `/v4`, … until the proxy returns a non-200 response.
4. Return the module path and version of the highest semver found.

The result — module path **and** version — is written back into both `opts.k6repo` and `opts.k6version`.

## Algorithm B: explicit version provided

Implemented in `sync.ResolveK6ModuleForVersion`.

Only `opts.k6repo` is updated here; the user-supplied version string is kept as-is.

### Case 1: clean semver tag (no prerelease suffix)

Examples: `v0.55.0`, `v2.0.0`, `v3.1.2`

The module path is inferred directly from the major version component using `semver.Major`:

- Major `v0` or `v1` → `go.k6.io/k6`
- Major `v2` → `go.k6.io/k6/v2`
- Major `vN` → `go.k6.io/k6/vN`

No network calls are made.

### Case 2: SHA, branch name, or pseudo-version

Examples: `d2c34d930734`, `master`, `v1.7.1-0.20260401084312-d2c34d930734`

These are not clean semver tags, so the module path cannot be inferred without querying the proxy. `probeK6ModuleForVersion` runs a two-step lookup:

#### Step 1 — resolve via `.info`

```
GET /go.k6.io/k6/@v/<ref>.info
```

The proxy accepts the raw SHA or branch name here and returns the canonical pseudo-version, e.g.:

```json
{"Version":"v1.7.1-0.20260401084312-d2c34d930734","Time":"2026-04-01T08:43:12Z"}
```

#### Step 2 — fetch `.mod` with canonical version

```
GET /go.k6.io/k6/@v/v1.7.1-0.20260401084312-d2c34d930734.mod
```

The `module` directive inside the returned `go.mod` is authoritative. If the commit belongs to the v2 module, the file will contain:

```
module go.k6.io/k6/v2
```

That declared path is returned — regardless of which module path was used to fetch it.

#### Fallback: try `/v2`, `/v3`, …

If the base module's `.info` returns a 404 (the proxy does not serve the reference under `go.k6.io/k6`), the same two-step lookup is repeated for `go.k6.io/k6/v2`, then `/v3`, and so on. The loop stops on the first failure, since a missing major version implies no higher major will have the reference either.

## Debug logging

All proxy URLs and resolution decisions are logged at `slog.Debug` level and are visible when `--verbose` / `-v` is passed:

```
DEBUG Resolving k6 module from extension dependencies count=1
DEBUG Checking extension go.mod for k6 dependency module=github.com/grafana/xk6-sql
DEBUG Found k6 dependency in extension extension=github.com/grafana/xk6-sql k6module=go.k6.io/k6/v2 k6version=v2.0.0
DEBUG Resolved k6 module repo=go.k6.io/k6/v2 version=v2.0.0
DEBUG Go proxy request url=https://proxy.golang.org/go.k6.io/k6/v2/@v/v2.0.0.mod
DEBUG Go proxy response url=https://proxy.golang.org/go.k6.io/k6/v2/@v/v2.0.0.mod status=200
```

## Decision flowchart

```
resolveK6Repo
│
├─ --k6-repo set explicitly? ──────────────────────── skip, use as-is
│
├─ --k6-version == "latest" (default)?
│   │
│   ├─ for each --with extension:
│   │   ├─ local replace path? ── read go.mod from disk
│   │   └─ remote?             ── fetch /@latest then /@v/<ver>.mod
│   │   └─ has k6 require?     ── return that path + version ✓
│   │
│   └─ no extension declared k6:
│       probe go.k6.io/k6/@latest, /v2/@latest, /v3/@latest …
│       return path with highest semver ✓
│
└─ explicit --k6-version?
    │
    ├─ clean semver (v2.0.0) ── infer from major, no network ✓
    │
    └─ SHA / branch / pseudo-version:
        ├─ GET go.k6.io/k6/@v/<ref>.info        (resolves to pseudo-version)
        │   GET go.k6.io/k6/@v/<pseudo>.mod     (read module declaration)
        │   declared path is authoritative ✓
        │
        └─ 404? try go.k6.io/k6/v2/@v/<ref>.info → .mod ✓
```
