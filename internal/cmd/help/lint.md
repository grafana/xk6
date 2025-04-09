Static analyzer for k6 extensions

**Linter for k6 extensions**

xk6 lint analyzes the source of the k6 extension and try to build k6 with the extension.

The contents of the source directory are used for analysis. If the directory is a git workdir, it also analyzes the git metadata. The analysis is completely local and does not use external APIs (e.g. repository manager API) or services.

The result of the analysis is compliance expressed as a percentage (`0`-`100`). This value is created as a weighted, normalized value of the scores of each checker. A compliance grade is created from the percentage value (`A`-`F`).

By default, text output is generated. The `--json` flag can be used to generate the result in JSON format.

If the grade is `C` or higher, the command is successful, otherwise it returns an exit code larger than `0`.
This passing grade can be modified using the `--passing` flag.
