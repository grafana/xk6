Build a custom k6 executable

This command can be used to build custom k6 executables with or without extensions.

The target platform (operating system, architecture) can be specified with flags or environment variables.

The k6 version to be used and the k6 repository (for forks) can be specified with flags or environment variables.

**Precedence**

If a setting can be specified with both a flag and an environment variable, the flag takes precedence.

**Extensions**

The `--with` flag can be used to specify one or more extensions to be included. Extensions can be referenced with the go module path, optionally followed by a version specification. In the case of a fork, the path of the forked go module can be specified as replacement.

**Fork**

The `--replace` flag can be used to specify a replacement for any go module. This allows forks to be used instead of extension dependencies.

A k6 fork can be specified with the `--k6-repo` flag (or the `K6_REPO` environment variable).

**k6 major version auto-resolution**

When `--k6-repo` is not set, xk6 automatically determines the correct k6 module path (`go.k6.io/k6`, `go.k6.io/k6/v2`, etc.) so that builds continue to work as k6 adopts higher major versions. The resolution strategy depends on what `--k6-version` is set to:

- **No version specified (`latest`):** xk6 inspects the `go.mod` of each `--with` extension to find which k6 major version they depend on. The first extension that declares a k6 dependency wins. If no extension declares k6, the Go proxy is queried for `go.k6.io/k6/@latest`, `/v2/@latest`, `/v3/@latest`, … and the module with the highest published version is used.

- **Clean semver tag (e.g. `v2.0.0`):** the module path is inferred directly from the major version component — no network calls required.

- **SHA, branch name, or pseudo-version:** xk6 uses a two-step Go proxy lookup to find which major-version module the reference belongs to. See [k6 module resolution](k6-module-resolution.md) for the full algorithm.

Pass `--verbose` to log every proxy request and resolution decision.
