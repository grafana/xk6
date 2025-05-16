Adjust example extension

This command will make the necessary adjustments after forking the example repository.

- The go module path and package name will be set
- The JavaScript import path will be set
- The example scripts will be adjusted to the new import path
- The extension will be ready to build and test

This is an internal command designed to be run as a Dev Container lifecycle script.

The git remote `origin` must be set or the command will skip the adjustment.
The changes will not be committed, they will only be applied to the working directory.

The folder containing the git working directory can be passed as an optional argument.
The default is the current folder.

An optional extension description can be specified as a flag.
The default description is generated from the go module path as follows:
- remote git URL is generated from the go module path
- the description is retrieved from the remote repository manager

An optional go package name can be specified as a flag.
The default go package name is generated from the go module path as follows:
- the last element of the go module path is kept
- the `xk6-output-` and `xk6-` prefixes are removed
- the `-` characters are replaced with `_` characters
