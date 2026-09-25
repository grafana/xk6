Installation

If you want to develop locally, outside a dev container, you'll need a Go toolchain either way, so installing xk6 as a package that brings Go along is usually the simplest option.

**Windows**

Install with [WinGet], which installs Go along with it:

    winget install GrafanaLabs.xk6

**macOS or Linux**

Install with [Homebrew], which installs Go along with it:

    brew install xk6

**Other platforms, or installing xk6 into an existing Go setup**

If you already have a [stable version] of Go installed, or your platform isn't covered by WinGet or Homebrew, install xk6 with:

    go install go.k6.io/xk6@latest

This installs the `xk6` binary in the `$GOPATH/bin` directory.

Precompiled binaries are also available from the [Releases] page.

[Homebrew]: https://brew.sh/
[Releases]: https://github.com/grafana/xk6/releases
[stable version]: https://go.dev/dl/
[WinGet]: https://learn.microsoft.com/windows/package-manager/winget/
