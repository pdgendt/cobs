/*
Encode reads from standard input, and writes the encoded data to standard output.

Usage:

	encode [flags]

The flags are:

	-d/-del
	    Append the encoded data with the sentinel delimiter.

	-s/-sentinel
	    Use a custom sentinel value (default is 0x00).
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

var appendDelim bool
var sentinel int

func init() {
	flag.BoolVar(&appendDelim, "del", false, "Append a delimiter")
	flag.BoolVar(&appendDelim, "d", false, "Append a delimiter")
	flag.IntVar(&sentinel, "sentinel", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
	flag.IntVar(&sentinel, "s", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
}

func main() {
	flag.Parse()

	if sentinel < 0 || sentinel > 255 {
		fmt.Fprintf(os.Stderr, "Error: sentinel value (%d) must be in [0x00, 0xFF]\n",
			sentinel)
		os.Exit(1)
	}

	enc := cobs.NewEncoder(os.Stdout, cobs.WithSentinel(byte(sentinel)))

	if _, err := io.Copy(enc, os.Stdin); err != nil {
		panic(err)
	}

	if err := enc.Close(); err != nil {
		panic(err)
	}

	if appendDelim {
		os.Stdout.Write([]byte{byte(sentinel)})
	}
}
