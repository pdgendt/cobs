/*
Decode reads from standard input, and writes the decoded data to standard output.

Usage:

    decode

When decode reads a zero delimiter it will stop processing data. If malformed encoded data
is passed the program will panic.
*/
package main

import (
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

func main() {
	dec := cobs.NewDecoder(os.Stdout)

	if _, err := io.Copy(dec, os.Stdin); err != nil && err != cobs.EOD {
		panic(err)
	}
}
