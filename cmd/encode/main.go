/*
Encode reads from standard input, and writes the encoded data to standard output.

Usage:

    encode [flags]

The flags are:

    -del
        Append the encoded data with a (zero) delimiter.
*/
package main

import (
	"flag"
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

func main() {
	delimiter := flag.Bool("del", false, "Append a delimiter")
	flag.Parse()

	enc := cobs.NewEncoder(os.Stdout)

	if _, err := io.Copy(enc, os.Stdin); err != nil {
		panic(err)
	}

	if err := enc.Close(); err != nil {
		panic(err)
	}

	if *delimiter {
		os.Stdout.Write([]byte{cobs.Delimiter})
	}
}
