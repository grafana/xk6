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
