Checkers

Compliance with the requirements expected of k6 extensions is checked by various compliance checkers. The result of the checks is compliance as a percentage value (`0-100`). This value is created as a weighted, normalized value of the scores of each checker. A compliance grade is created from the percentage value (`A`-`F`, `A` is the best).

- `security` - check for security issues (using the `gosec` tool)
- `vulnerability` - check for vulnerability issues (using the `govulncheck` tool)
- `module` - checks if there is a valid `go.mod`
- `replace` - checks if there is no `replace` directive in `go.mod`
- `readme` - checks if there is a readme file
- `examples` - checks if there are files in the `examples` directory
- `license` - checks whether there is a suitable OSS license
- `git` - checks if the directory is git workdir
- `versions` - checks for semantic versioning git tags
- `build` - checks if the latest k6 version can be built with the extension
- `smoke` - checks if the smoke test script exists and runs successfully (`smoke.js`, `smoke.ts`, `smoke.test.js` or `smoke.test.ts` in the `test`,`tests`, `examples`, `scripts` or in the base directory)
- `types` - checks if the TypeScript API declaration file exists (`index.d.ts` in the `docs`, `api-docs` or the base directory)
- `codeowners` - checks if there is a `CODEOWNERS` file (for official extensions) (in the `.github` or `docs` or in the base directory)
