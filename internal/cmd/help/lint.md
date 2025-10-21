Analyze k6 extension compliance

Validate k6 extension source code against quality, security, and compatibility standards.
Performs static analysis, builds the extension with k6, and checks compliance requirements.

Use presets to run predefined sets of checks, or customize with individual checkers.
The analysis is performed locally using the source directory contents and Git metadata.

Exit Codes:
  - `0`   All checks passed
  - `1`   Unexpected execution error
  - `2`   One or more checks failed
