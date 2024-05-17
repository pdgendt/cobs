package main

import (
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

func main() {
	dec := cobs.NewDecoder(os.Stdout)
	for {
		tmp := make([]byte, 1024)
		n, err := os.Stdin.Read(tmp)
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		
		_, err = dec.Write(tmp[:n])
		if err != nil {
			panic(err)
		}
	}

}
