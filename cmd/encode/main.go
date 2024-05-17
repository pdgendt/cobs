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
	for {
		tmp := make([]byte, 1024)
		n, err := os.Stdin.Read(tmp)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		
		_, err = enc.Write(tmp[:n])
		if err != nil {
			panic(err)
		}
	}
	enc.Close()

	if *delimiter {
		os.Stdout.Write([]byte{cobs.Delimiter})
	}
}
