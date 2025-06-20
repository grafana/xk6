Synchronize dependencies with k6

Synchronizes the versions of dependencies in `go.mod` with those used in the k6 project. Dependencies not found in k6's `go.mod` remain unchanged. Future updates may include synchronization of other files.

The purpose of this subcommand is to avoid dependency conflicts when building the extension with k6 (and other extensions).
