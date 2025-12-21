# COBS

[![Go version](https://img.shields.io/github/go-mod/go-version/pdgendt/cobs)](https://pkg.go.dev/github.com/pdgendt/cobs)
[![Code coverage](https://codecov.io/github/pdgendt/cobs/graph/badge.svg?token=CMGPE2NO89)](https://codecov.io/github/pdgendt/cobs)

A Golang module for the
[Consistent Overhead Byte Stuffing (COBS)](https://en.wikipedia.org/wiki/Consistent_Overhead_Byte_Stuffing)
algorithm.

## Usage

This module provides both simple helper functions to Encode/Decode `[]byte` arrays and
Encoder/Decoder structs to stream bytes to an `io.Writer` instance.

### `Encode`/`Decode` functions

The helper functions will allocate buffers to hold the encoded/decoded data and return a `[]byte`
slice or an error.

The following example encodes a string with embedded zeroes:

```go
package main

import (
	"os"

	"github.com/pdgendt/cobs"
)

func main() {
	enc, _ := cobs.Encode([]byte{'H', 'e', 'l', 'l', 'o', 0x00, 'w', 'o', 'r', 'l', 'd', '!'})

	os.Stdout.write(enc)
}
```

### `Encoder`/`Decoder` structs

The structs require an `io.Writer` instance on creation. As soon as data is available it is written,
for the `Encoder` this is done for each group, with a maximum of 255 bytes, for the `Decoder` every
`byte` is written individually.

The structs implement the `io.Writer` interface to allow chaining.

The following example encodes bytes read from `os.Stdin` and writes them to `os.Stdout`:

```go
package main

import (
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

func main() {
	enc := cobs.NewEncoder(os.Stdout)

	if _, err := io.Copy(enc, os.Stdin); err != nil {
		panic(err)
	}

	// Close needs to be called to flush the last group
	if err := enc.Close(); err != nil {
		panic(err)
	}
}
```

### Sentinel Value

By default, COBS uses `0x00` (zero) as the sentinel value - the special byte that never
appears in encoded data and is used for packet framing. You can configure a custom sentinel
value using the `WithSentinel()` option. This is useful when your protocol already uses
zero bytes, or when you need a different delimiter for packet framing.

When a custom sentinel value is used, the implementation applies an XOR operation with
the sentinel value on the encoded data after encoding and before decoding. This transforms
the standard COBS delimiters (0x00) to the custom sentinel value, allowing any byte to be
used as the packet delimiter.

```go
// Use 0xFF as the sentinel instead of 0x00
enc := cobs.NewEncoder(w, cobs.WithSentinel(0xFF))
dec := cobs.NewDecoder(w, cobs.WithSentinel(0xFF))

// Or with the helper functions
encoded, _ := cobs.Encode(data, cobs.WithSentinel(0xFF))
decoded, _ := cobs.Decode(encoded, cobs.WithSentinel(0xFF))
```

### COBS/R (COBS Reduced)

COBS/R is a variant of the COBS encoding that provides slightly better encoding efficiency
by saving one byte in certain cases. In standard COBS, the overhead byte at the start of each
group indicates where the next zero byte would have been. In COBS/R, if the overhead byte value
is less than the last data byte in a group, the overhead byte is replaced with that last data byte,
and the last data byte is omitted. This can save one byte when encoding data that ends with a
byte value larger than the group size.

You can enable COBS/R using the `WithReduced()` option:

```go
// Use COBS/R encoding
enc := cobs.NewEncoder(w, cobs.WithReduced(true))
dec := cobs.NewDecoder(w, cobs.WithReduced(true))

// Or with the helper functions
encoded, _ := cobs.Encode(data, cobs.WithReduced(true))
decoded, _ := cobs.Decode(encoded, cobs.WithReduced(true))
```

COBS/R can be combined with custom sentinel values:

```go
// Use both COBS/R and a custom sentinel
encoded, _ := cobs.Encode(data, cobs.WithReduced(true), cobs.WithSentinel(0xFF))
decoded, _ := cobs.Decode(encoded, cobs.WithReduced(true), cobs.WithSentinel(0xFF))
```

**Note:** When using COBS/R decoding, you must call `Close()` on the decoder to flush the final
byte, as the reduced encoding may hold back the last byte until it knows encoding is complete.

For more information about COBS/R, see the [Python COBS documentation](https://pythonhosted.org/cobs/cobsr-intro.html).

## CLI tools

The [cmd/](cmd/) directory contains simple encode/decode command line tools that take in data
from `stdin` and writes it to `stdout`.

This can be used to pipe encoded/decoded data to other processes.

```shell
$ echo "Hello world" | go run cmd/encode/main.go | go run cmd/decode/main.go
Hello world
```

### Custom Sentinel in CLI

Both `encode` and `decode` commands support the `-s` (or `-sentinel`) flag to specify a custom
sentinel value:

```shell
$ echo "Hello world" | go run cmd/encode/main.go -s 0xFF | go run cmd/decode/main.go -s 0xFF
Hello world
```

The `encode` command also supports the `-d` (or `-del`) flag to append the sentinel delimiter
after the encoded data.

### COBS/R in CLI

Both `encode` and `decode` commands support the `-r` (or `-reduced`) flag to enable COBS/R encoding:

```shell
$ echo "Hello world" | go run cmd/encode/main.go -r | go run cmd/decode/main.go -r
Hello world
```

The `-r` flag can be combined with custom sentinel values:

```shell
$ echo "Hello world" | go run cmd/encode/main.go -r -s 0xFF | go run cmd/decode/main.go -r -s 0xFF
Hello world
```
