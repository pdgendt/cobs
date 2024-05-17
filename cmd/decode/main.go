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
