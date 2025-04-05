Execute the run command with the custom k6

This is a useful command when developing the k6 extension. After modifying the source code of the extension, a k6 test script can simply be run without building the k6 executable.

Under the hood, the command builds a k6 executable into a temporary directory and runs it with the arguments. The usual flags for the build command can be used.

Two dashes are used to indicate that the following flags are no longer the flags of the `xk6 run` command but the flags of the `k6 run` command.
