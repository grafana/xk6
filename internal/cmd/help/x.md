Execute a k6 subcommand provided by the current directory's extension

This command is useful when developing k6 subcommand extensions. After modifying the extension source code in the current directory, you can execute the subcommand directly without manually building the k6 executable.

Under the hood, xk6 builds a temporary k6 executable with your extensions and runs it with the provided arguments. All standard build command flags are supported.

Use two dashes (`--`) to separate xk6 flags from k6 subcommand flags.
