[![Go Reference](https://pkg.go.dev/badge/github.com/szkiba/efa.svg)](https://pkg.go.dev/github.com/szkiba/efa)
[![Validate](https://github.com/szkiba/efa/actions/workflows/validate.yml/badge.svg)](https://github.com/szkiba/efa/actions/workflows/validate.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/szkiba/efa)](https://goreportcard.com/report/github.com/szkiba/efa)
[![Codecov](https://codecov.io/gh/szkiba/efa/graph/badge.svg?token=tTOQkfVcdB)](https://codecov.io/gh/szkiba/efa)

# ｅｆａ

**Set go flag values ​​from the environment**

The **efa** library enables setting go flag values ​​from environment variables. The name of the environment variable can be generated from the flag name (and an optional prefix) or can be specified directly.

### Features

- supports the standard go [flag package](https://pkg.go.dev/flag)
- supports the [spf13/pflag](https://github.com/spf13/pflag) package
- can be used with the [spf13/cobra](https://github.com/spf13/cobra) package
- supports prefixed environment variables
- can automatically prefix from the executable name
- annotate flags with the environment variable name
- has no dependencies

### Usage

```go file=example_bind_linux_test.go
package efa_test

import (
	"flag"
	"fmt"
	"os"

	"github.com/szkiba/efa"
)

// To emulate the "EFA_TEST_QUESTION='To be, or not to be?'" shell command.
func init() {
	// Note: the go test framework sets the executable name to "efa.test".
	os.Setenv("EFA_TEST_QUESTION", "To be, or not to be?")
}

func ExampleBind() {
	question := flag.String("question", "How many?", "The question")

	// Must be called before parsing!
	efa.Bind("question")

	flag.Parse()

	fmt.Println(*question)

	// Output:
	// To be, or not to be?
}
```

More examples can be found in the `example*_test.go` files.

### Status

The initial implementation is complete. Although **efa** is still in a relatively early stage of development, but it is already usable.
