Static analyzer for k6 extensions

**Linter for k6 extensions**

xk6 lint analyzes the source of the k6 extension and try to build k6 with the extension.

The contents of the source directory are used for analysis. If the directory is a git workdir, it also analyzes the git metadata. The analysis is completely local and does not use external APIs (e.g. repository manager API) or services.

Compliance with the requirements expected of k6 extensions is checked by various compliance checkers.

- `module` - checks if there is a valid `go.mod`
- `replace` - checks if there is no `replace` directive in `go.mod`
- `readme` - checks if there is a readme file
- `examples` - checks if there are files in the `examples` directory
- `license` - checks whether there is an acceptable license
  - `AGPL-3.0`
  - `Apache-2.0`
  - `BSD`
  - `GPL-3.0`
  - `LGPL-3.0`
  - `MIT`
- `git` - checks if the directory is git workdir
- `versions` - checks for semantic versioning git tags
- `build` - checks if the latest k6 version can be built with the extension
- `smoke` - checks if the smoke test script exists and runs successfully (`smoke.js`, `smoke.ts`, `smoke.test.js` or `smoke.test.ts` in the `test`,`tests`, `examples` or the base directory)
- `types` - checks if the TypeScript API declaration file exists (`index.d.ts` in the `docs`, `api-docs` or the base directory)
- `codeowners` - checks if there is a CODEOWNERS file (for official extensions) (in the `.github` or `docs` or in the base directory)

The result of the analysis is compliance expressed as a percentage (`0`-`100`). This value is created as a weighted, normalized value of the scores of each checker. A compliance grade is created from the percentage value (`A`-`F`).

By default, text output is generated. The `--json` flag can be used to generate the result in JSON format.

If the grade is `C` or higher, the command is successful, otherwise it returns an exit code larger than `0`.
This passing grade can be modified using the `--passing` flag.
