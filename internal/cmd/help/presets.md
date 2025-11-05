Available Presets

The following presets are available for use with the `xk6 lint` command.

#### `all`

Comprehensive preset that includes every available check in the xk6 linting system. Serves as a complete reference for all possible compliance checks and provides maximum validation coverage for development and testing purposes.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`
  - `codeowners`

#### `loose`

Minimal preset focusing on essential quality and security compliance checks. Designed for development environments and initial extension development phases. Provides basic compliance requirements without restrictive validation that slows development cycles. This is the default preset.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`

#### `strict`

Comprehensive preset for production-ready extensions, including all compliance checks except those reserved for official Grafana extensions (such as codeowners validation). Designed for third-party extensions that require high quality standards before release.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`

#### `private`

Lightweight preset designed for private or internal extension development. Focuses on core security and functionality compliance while omitting documentation and public-facing requirements such as README formatting and licensing compliance.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `git`

#### `community`

Balanced preset tailored for community-contributed extension development. Includes essential quality, security, and documentation compliance to ensure extensions meet community standards while remaining accessible to contributors.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`

#### `official`

Most stringent preset for official Grafana-maintained extension development. Enforces the highest quality standards including code ownership compliance, comprehensive testing requirements, and complete documentation compliance.

Included Checks:
  - `security`
  - `vulnerability`
  - `module`
  - `replace`
  - `readme`
  - `license`
  - `git`
  - `versions`
  - `build`
  - `smoke`
  - `examples`
  - `types`
  - `codeowners`

