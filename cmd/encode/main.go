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
