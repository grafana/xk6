Available Checks

The following checks are available for use with the `xk6 lint` command.

**`security`**

Performs static security analysis on Go source code using the `gosec` tool to identify potential security vulnerabilities, insecure coding patterns, and compliance violations.

_Security vulnerabilities in extensions can compromise the entire k6 testing environment and potentially expose sensitive data or system resources. Early detection of security flaws through static analysis helps maintain the integrity of the k6 ecosystem and protects users from malicious or poorly secured extensions._

Resolution

Install `gosec` with `go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest`, then run `gosec ./...` to scan your codebase. Address all HIGH and MEDIUM severity findings by following secure coding practices, input validation, and proper error handling. Consider adding `// #gosec` comments only for verified false positives with clear justification.

**`vulnerability`**

Scans for known security vulnerabilities in Go modules and their dependencies using the official `govulncheck` tool from the Go security team.

_Third-party dependencies often contain discovered vulnerabilities that could be exploited in production environments. This check ensures that extensions don't introduce known security risks through outdated or vulnerable dependencies, maintaining the security posture of k6 installations._

Resolution

Install `govulncheck` with `go install golang.org/x/vuln/cmd/govulncheck@latest`, then run `govulncheck ./...` to scan for vulnerabilities. Update vulnerable dependencies to patched versions using `go get -u package@version`. If no patch is available, consider alternative packages or implement additional security measures.

**`module`**

Validates the presence and structure of a `go.mod` file, ensuring proper module declaration, Go version compatibility, and dependency specifications.

_A properly configured `go.mod` file is fundamental for Go module system functionality, enabling reproducible builds, version management, and dependency resolution. Without it, the extension cannot be properly integrated into the k6 build process or distributed through Go's module system._

Resolution

Create a `go.mod` file in the extension root using `go mod init github.com/your-org/your-extension`, ensuring the Go version is specified as `go 1.23` (or appropriate minimum version). Run `go mod tidy` to populate dependencies and remove unused ones, then verify the module path matches your repository structure.

**`replace`**

Detects and flags any `replace` directives in the `go.mod` file that could cause dependency resolution issues or prevent proper extension distribution.

_Replace directives create local overrides that only work in the development environment and break when the extension is built by xk6 or distributed to users. They can mask dependency conflicts, create irreproducible builds, and prevent proper version resolution in the broader Go ecosystem._

Resolution

Remove all `replace` directives from `go.mod`. If you need to use a fork or modified dependency, publish it as a proper Go module with a different import path. For local development, consider using `go work` workspaces instead of replace directives, or contribute fixes upstream to the original repository.

**`readme`**

Verifies the existence of a README file in standard formats (Markdown, text, AsciiDoc, etc.) that provides essential information about the extension.

_A comprehensive README serves as the primary documentation entry point, helping users understand the extension's purpose, installation process, usage examples, and contribution guidelines. It significantly impacts adoption rates and reduces support burden by providing self-service information for common questions._

Resolution

Create a `README.md` file in the extension root directory containing extension description and purpose, installation instructions via xk6, and usage examples with sample k6 scripts. Include API documentation or links to detailed docs, contributing guidelines and development setup, and license information and acknowledgments.

**`license`**

Validates that the extension includes a recognized open-source license file compatible with the k6 ecosystem and Go module distribution requirements.

_A clear license is legally required for code distribution and defines usage rights for users, contributors, and organizations. Without proper licensing, extensions cannot be safely used in commercial environments or contributed to by the community. Accepted licenses ensure compatibility with k6's Apache 2.0 license._

Resolution

Add a `LICENSE` file to the repository root with one of the approved licenses: MIT (recommended for maximum compatibility), Apache-2.0 (best for corporate environments), BSD-2-Clause or BSD-3-Clause, or GPL-3.0, LGPL-3.0, or AGPL-3.0 (for copyleft requirements).

**`git`**

Verifies that the extension directory is a valid Git repository with proper version control initialization and configuration.

_Git version control is essential for extension development, enabling change tracking, collaboration, release management, and integration with Go's module system which relies on Git tags for versioning. Extensions without Git cannot be properly distributed or versioned through standard Go tooling._

Resolution

Initialize Git in the extension directory with `git init`, add a `.gitignore` file appropriate for the extension, then stage and commit all extension files using `git add . && git commit -m "Initial commit"`. Consider setting up a remote repository on GitHub, GitLab, or similar platform for collaboration and distribution.

**`versions`**

Validates the presence of proper semantic versioning Git tags following the vMAJOR.MINOR.PATCH format required by Go modules and xk6.

_Semantic versioning tags are critical for Go module resolution, allowing users to specify version constraints and enabling automatic dependency management. Proper versioning communicates API compatibility, helps users understand upgrade risks, and enables tools like Dependabot to manage updates automatically._

Resolution

Create an initial release tag using `git tag v0.1.0 && git push origin v0.1.0`. For future releases, increment versions appropriately: PATCH (v1.0.1) for bug fixes with no API changes, MINOR (v1.1.0) for new features that are backward compatible, and MAJOR (v2.0.0) for breaking changes or API modifications. Always follow semantic versioning principles for predictable dependency management.

**`build`**

Performs a complete build test of the extension using xk6 with the latest stable k6 version to verify compilation and linking compatibility.

_Build compatibility is essential for user adoption and long-term maintainability. This check catches compilation errors, API compatibility issues, and dependency conflicts that would prevent users from successfully building custom k6 binaries with the extension, ensuring a smooth user experience._

Resolution

Test the build locally using `xk6 build --with github.com/your-org/your-extension@latest`, then fix any compilation errors, missing imports, or API incompatibilities. Ensure your extension properly implements required k6 extension interfaces and update dependencies if needed using `go get -u && go mod tidy`.

**`smoke`**

Locates and executes a smoke test script to verify basic extension functionality works correctly in a real k6 runtime environment.

_Smoke tests provide essential validation that the extension's core functionality operates as expected when loaded into k6. They catch runtime errors, API mismatches, and integration issues that static analysis cannot detect, serving as the minimum viable test to ensure the extension actually works for end users._

Resolution

Create a smoke test file as `smoke.js` or `smoke.ts` in root, `test/`, `tests/`, or `examples/` directory, including basic functionality tests that import the extension and call main functions. Ensure the test runs without errors when executed with your custom k6 build.

**`examples`**

Ensures the presence of an `examples/` directory containing practical k6 scripts that demonstrate the extension's functionality and usage patterns.

_Example scripts are crucial for user onboarding and adoption, providing immediate practical value and reducing the learning curve. They serve as living documentation, showing real-world usage patterns and helping users quickly understand how to integrate the extension into their testing workflows._

Resolution

Create an `examples/` directory with multiple k6 JavaScript/TypeScript files including a basic usage example showing core functionality, advanced example demonstrating complex features, and integration examples with other k6 features. Include comments explaining key concepts and parameters. Add a README.md in examples/ explaining how to run each script.

**`types`**

Validates the presence of TypeScript declaration files (`index.d.ts`) that define the extension's API surface and enable type-safe usage in TypeScript k6 scripts.

_TypeScript declarations significantly improve developer experience by providing IDE autocompletion, type checking, and inline documentation. As k6 increasingly supports TypeScript, providing accurate type definitions becomes essential for extension adoption and proper integration with modern development workflows._

Resolution

Create an `index.d.ts` file in the root, `docs/`, or `api-docs/` directory defining all exported functions, classes, and interfaces with parameter types and return types. Include JSDoc comments for function documentation and proper module declarations matching your extension's import path.

**`codeowners`**

Validates the existence of a `CODEOWNERS` file that defines maintainership responsibilities and automated review assignments for different parts of the codebase.

_Code ownership is critical for maintaining extension quality and ensuring timely responses to issues and pull requests. CODEOWNERS enables automatic reviewer assignment, helps contributors identify the right people for questions, and establishes clear accountability for different components of the extension._

Resolution

Create a `CODEOWNERS` file in `.github/`, `docs/`, or the repository root defining global owners (`* @username @team`) and specifying path-based ownership (`docs/ @doc-team`). Include email contacts for critical components and use GitHub teams when possible for better maintainability. Ensure all specified owners have appropriate repository permissions.

