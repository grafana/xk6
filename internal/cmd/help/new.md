Create a new k6 extension

Create and initialize a new k6 extension using one of the predefined templates.

The go module path of the new extension must be passed as an argument.

An optional extension description can be specified as a flag.
The default description is generated from the go module path as follows:
- remote git URL is generated from the go module path
- the description is retrieved from the remote repository manager

An optional go package name can be specified as a flag.
The default go package name is generated from the go module path as follows:
- the last element of the go module path is kept
- the `xk6-output-` and `xk6-` prefixes are removed
- the `-` characters are replaced with `_` characters

A JavaScript type k6 extension will be generated by default.
The extension type can be optionally specified as a flag.

The `grafana/xk6-example` and `grafana/xk6-output-example` GitHub repositories are used as sources for generation.
