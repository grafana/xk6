# ｄｏｃｓｍｅ

**Keep README up to date based on CLI help**

The **docsme** go library makes it possible to automatically keep the documentation of command line tools up-to-date based on their CLI help.  Markdown documentation can be generated from [Cobra](https://github.com/spf13/cobra) command definitions. The generated documentation can be used to update part of `README.md` or can be written to a separate file.

### Features

- documentation and CLI help are guaranteed to be in sync
- documentation can be kept up to date automatically
- used only at build time, does not increase the size of the CLI tool
- gives motivation to write good CLI help

### Usage

**As a build-time tool.**

The advantage of using **docsme** as a build-time tool is that the **docsme** library is not built into the CLI at all.

To use it as a build-time tool, a `tools/docsme/main.go` file should be created with content similar to the following:

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/szkiba/docsme"
)

// newCommand is a factory function for generating the root cobra.Command.
func newCommand() *cobra.Command {
    root := &cobra.Command {
        // command definition goes here
    }

    // add subcommands here

    return root
}

func main() {
    cobra.CheckErr(docsme.For(newCommand()).Execute())
}
```

Help is available with the usual `--help` flag.

```bash
go run ./tools/docsme --help
```

**As a subcommand**

The advantage of using **docsme** as a subcommand is that documentation can be generated at any time with the CLI tool. Since **docsme** is a small library without additional dependencies, embedding it does not increase the size of the CLI.

To use it as a subcommand, the **docsme** Cobra command definition should be added to the CLI command.

```go
package main

import (
    "github.com/spf13/cobra"
    "github.com/szkiba/docsme"
)

// newCommand is a factory function for generating the root cobra.Command.
func newCommand() *cobra.Command {
    root := &cobra.Command {
        Use: "mycli",
        // command definition goes here
    }

    root.AddCommand(docsme.New())

    return root
}

func main() {
    cobra.CheckErr(newCommand().Execute())
}
```

The `docsme.New()` factory function creates the Cobra command definition with the name `docs`. Help is available from the `docs` subcommand with the `--help` flag.

```bash
mycli docs --help
```

### Status

Following [Readme Driven Development](https://tom.preston-werner.com/2010/08/23/readme-driven-development.html), the README of the **docsme** library was created first.

The initial implementation is complete. Although **docsme** is still in a relatively early stage of development, but it is already usable.

## CLI Reference

This section contains the reference documentation for the command line interface.

This is actually a dogfooding, because **docsme** generates its own documentation with the `tools/docsme/main.go` build-time tool:

```go file=tools/docsme/main.go
// Package main contains docsme's own build-time documentation generation tool.
package main

import (
	"github.com/spf13/cobra"
	"github.com/szkiba/docsme"
)

func main() {
	cobra.CheckErr(docsme.For(nil).Execute())
}
```

<!-- #region cli -->
## docsme

Keep documentation up to date based on CLI help

### Synopsis

Generate CLI documentation based on Cobra command definitions. The generated documentation can replace a region of the output file (or overwrite the entire file).

If the Cobra command contains subcommands, they will also be traversed recursively and their documentation will be included in the output.

**Build-time tool**

A typical use of docsme is to create a build-time tool in the `tools/docsme/main.go` file with content similar to:

    func main() {
        cobra.CheckErr(docsme.For(newCommand()).Execute())
    }

Where the `newCommand()` factory function generates the root command of the CLI tool.

This build-time tool can be run with the `go run` command to update the documentation:

    go run ./tools/docsme [flags]

**Regions**

A part of the output file can be marked with a so-called region comment.

    <!-- #region NAME -->
    ...
    <!-- #endregion NAME -->

The documentation in the region marked in this way can be updated using the `--region` flag. The default region name is `cli`.

**Update README.md**

A typical use of docsme is to update the CLI documentation in the `README.md` file. In the `README.md` file, the CLI documentation is marked with a region comment (e.g. `cli` region) and is updated using the following command:

    go run ./tools/docsme -r cli -o README.md

When used without parameters, the generated documentation is written to standard output.

### Usage

```bash
docsme [flags]
```

### Flags

```
  -o, --output string   Output filename (default stdout)
  -r, --region string   File region to update (default "cli")
  -f, --force           Force file overwrite
      --heading int     Initial heading level (default 2)
  -h, --help            help for docsme
```

<!-- #endregion cli -->
