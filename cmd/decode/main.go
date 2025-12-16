/*
Decode reads from standard input, and writes the decoded data to standard output.

Usage:

	decode [flags]

The flags are:

	-s/-sentinel
	    Use a custom sentinel value (default is 0x00).

	-r/-reduced
	    Use COBS reduced (COBS/R).

When decode reads the sentinel delimiter it will stop processing data. If malformed encoded data
is passed the program will panic.
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

var sentinel int
var reduced  bool

func init() {
	flag.IntVar(&sentinel, "sentinel", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
	flag.IntVar(&sentinel, "s", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
	flag.BoolVar(&reduced, "reduced", false, "Use COBS reduced (COBS/R)")
	flag.BoolVar(&reduced, "r", false, "Use COBS reduced (COBS/R)")
}

func main() {
	flag.Parse()

	if sentinel < 0 || sentinel > 255 {
		fmt.Fprintf(os.Stderr, "Error: sentinel value (%d) must be in [0x00, 0xFF]\n",
			sentinel)
		os.Exit(1)
	}

	dec := cobs.NewDecoder(os.Stdout, cobs.WithSentinel(byte(sentinel)), cobs.WithReduced(reduced))

	if _, err := io.Copy(dec, os.Stdin); err != nil && err != cobs.EOD {
		panic(err)
	}

	if err := dec.Close(); err != nil {
		panic(err)
	}
}
