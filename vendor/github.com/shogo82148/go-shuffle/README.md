# go-shuffle

[![test](https://github.com/shogo82148/go-shuffle/actions/workflows/test.yml/badge.svg)](https://github.com/shogo82148/go-shuffle/actions/workflows/test.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/shogo82148/go-shuffle.svg)](https://pkg.go.dev/github.com/shogo82148/go-shuffle)

Package shuffle provides primitives for shuffling slices and user-defined collections.

```go
import "github.com/shogo82148/go-shuffle"

// Shuffle slice of int.
ints := []int{3, 1, 4, 1, 5, 9}
shuffle.Ints(ints)

// Shuffle slice of int64.
int64s := []int64{3, 1, 4, 1, 5, 9}
shuffle.Int64s(int64s)

// Shuffle slice of string.
strings := []string{"foo", "bar"}
shuffle.Strings(strings)

// Shuffle slice of float64.
float64s := []float64{3, 1, 4, 1, 5, 9}
shuffle.Float64s(float64s)

// Shuffle slices
shuffle.Slice(ints)
shuffle.Slice(int64s)
shuffle.Slice(strings)
shuffle.Slice(float64s)
```

See [godoc](https://pkg.go.dev/github.com/shogo82148/go-shuffle) for more information.

# LICENSE

This software is released under the MIT License, see LICENSE.md.
