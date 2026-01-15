Synchronize dependencies with k6

Synchronizes the versions of dependencies in `go.mod` with those used in the k6 project. Dependencies not found in k6's `go.mod` remain unchanged. Future updates may include synchronization of other files.

The purpose of this subcommand is to avoid dependency conflicts when building the extension with k6 (and other extensions).

It is recommended to keep dependencies in common with k6 core in the same version k6 core uses. This guarantees binary compatibility of the JS runtime, and ensures uses will not have to face unforeseen build-time errors when compiling several extensions together with xk6.

By default, `xk6 sync` uses the k6 version specified in `go.mod`. This allows using any version supported by the `go get` command, including branch names like `master`. The `-k` or `--k6-version` flag can override this to sync with a specific k6 version. In this case only immutable versions can be used and `latest` which refers to latest immutable version.
