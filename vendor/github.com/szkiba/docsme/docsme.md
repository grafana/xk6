Keep documentation up to date based on CLI help

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
