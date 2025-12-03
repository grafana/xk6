Run integration tests with the custom k6

This command is useful for testing k6 extensions during development. It builds k6 with the extension once and runs multiple test scripts, reporting test results based on exit codes.

Under the hood, the command builds a k6 executable into a temporary directory and executes each test script with it. The usual flags for the build command can be used.

**Output Format**

By default, test results are reported in TAP (Test Anything Protocol) format for easy parsing and integration with CI systems. Use the `--json` flag to generate a CTRF (Common Test Report Format) JSON file for structured test reporting.

**Exit Codes**

The command exits with:
- `0` if all tests pass
- `1` if a command error occurs (invalid arguments, build failure, etc.)
- `2` if one or more tests fail

**Test Scripts**

One or more test file patterns must be specified as arguments. 

A test passes if the k6 script exits with code 0, and fails otherwise. Tests can fail through:
- Failed checks with threshold configurations
- Calling the `test.fail()` API
- Using k6 jslib testing/assertion frameworks
- Any uncaught exception or k6-specific exit code (97-110)

**Glob Patterns**

Glob patterns are supported in filenames:
- Asterisk wildcards (`*`)
- Super-asterisk wildcards (`**`) for recursive directory matching
- Single symbol wildcards (`?`)
- Character list matchers with negation and ranges (`[abc]`, `[!abc]`, `[a-c]`)
- Alternative matchers (`{a,b}`)
- Nested globbing (`{a,[bc]}`)

**Examples**

    # Run a single test
    xk6 test tests/integration.js

    # Run multiple test files
    xk6 test tests/test1.js tests/test2.js

    # Use glob patterns (recursive)
    xk6 test "tests/**/*.test.js"

    # Multiple patterns
    xk6 test "tests/**/*.test.js" "integration/**/*.spec.js"

    # Generate CTRF JSON report
    xk6 test --json --output report.json "tests/**/*.js"

    # Verbose output for debugging
    xk6 test --verbose tests/integration.js

**Using Pre-built k6**

The `--k6` flag allows testing with a pre-built k6 binary instead of building from source. This is useful when the Go toolchain is not available or when k6 doesn't need to be rebuilt.

    # Use pre-built k6 binary
    xk6 test --k6 /path/to/k6 tests/integration.js
