Static analyzer for k6 extensions

**Linter for k6 extensions**

xk6 lint analyzes the source code of a k6 extension and attempts to build k6 with it. It checks for compliance with k6 extension requirements using various built-in checkers.

The analysis is local and uses the contents of the provided source directory. It also analyzes git metadata if the directory is a git workdir. No external APIs or services are used.

Compliance is checked by the following individual checkers:

* `security` - Check for security issues (using the `gosec` tool)
* `vulnerability` - Check for vulnerability issues (using the `govulncheck` tool)
* `module`: Checks for a valid `go.mod` file.
* `replace`: Ensures there are no `replace` directives in `go.mod`.
* `readme`: Checks for the presence of a readme file.
* `examples`: Verifies files exist in the `examples` directory.
* `license`: Checks for an acceptable license, including: `MIT`, `Apache-2.0`, `GPL-3.0`, `AGPL-3.0`, `LGPL-3.0`, and `BSD`.
* `git`: Confirms the source directory is a git workdir.
* `versions`: Checks for semantic versioning git tags.
* `build`: Attempts to build the extension with the latest k6 version.
* `smoke`: Checks for and successfully runs a smoke test script (`smoke.js`, `smoke.ts`, `smoke.test.js`, or `smoke.test.ts` in `test`, `tests`, `examples`, or base directory).
* `types`: Checks for the existence of a TypeScript API declaration file (`index.d.ts` in `docs`, `api-docs`, or base directory).
* `codeowners`: Checks for a `CODEOWNERS` file in the `.github`, `docs`, or base directory (for official extensions).

The overall result is the logical product of all individual checker results:

- The run **passes** if and only if **all** individual checkers pass.
- The run **fails** if **any** individual checker fails.

If the run passes, the command is successful and exits with code `0`. If the run fails, it returns an exit code greater than `0`.

By default, text output is generated. The `--json` flag can be used to generate the result in JSON format.

