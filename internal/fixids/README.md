# fixids

Fix user id and group id in docker containers.

Original source code is from [fixuid](https://github.com/boxboat/fixuid) (v0.6.0).

**Usage**

```bash
#!/bin/sh

set -e

eval "$(fixids -q xk6 xk6)"

exec xk6 "$@"
```
