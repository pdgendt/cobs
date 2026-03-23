/*
Cobs reads from standard input, and writes encoded or decoded data to standard output.

Usage:

	cobs <command> [flags]

The commands are:

	encode    Encode data using COBS
	decode    Decode COBS-encoded data

Common flags:

	-s/-sentinel
	    Use a custom sentinel value (default is 0x00).

	-r/-reduced
	    Use COBS reduced (COBS/R).

Encode flags:

	-d/-del
	    Append the encoded data with the sentinel delimiter.

When decode reads the sentinel delimiter it will stop processing data. If malformed encoded data
is passed the program will panic.
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pdgendt/cobs"
)

type commonFlags struct {
	sentinel int
	reduced  bool
}

func (f *commonFlags) register(fs *flag.FlagSet) {
	fs.IntVar(&f.sentinel, "sentinel", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
	fs.IntVar(&f.sentinel, "s", int(cobs.Delimiter), "Sentinel value (default is 0x00)")
	fs.BoolVar(&f.reduced, "reduced", false, "Use COBS reduced (COBS/R)")
	fs.BoolVar(&f.reduced, "r", false, "Use COBS reduced (COBS/R)")
}

func (f *commonFlags) validate() error {
	if f.sentinel < 0 || f.sentinel > 255 {
		return errors.New("sentinel value (%d) must be in [0x00, 0xFF]")
	}

	return nil
}

func main() {
	var encFlags, decFlags commonFlags

	encodeCmd := flag.NewFlagSet("encode", flag.ExitOnError)
	encFlags.register(encodeCmd)
	appendDelim := encodeCmd.Bool("del", false, "Append a delimiter")
	encodeCmd.BoolVar(appendDelim, "d", false, "Append a delimiter")

	decodeCmd := flag.NewFlagSet("decode", flag.ExitOnError)
	decFlags.register(decodeCmd)

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: cobs <encode|decode> [flags]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "encode":
		encodeCmd.Parse(os.Args[2:])
		if err := encFlags.validate(); err != nil {
			fatal(err)
		}

		enc := cobs.NewEncoder(
			os.Stdout,
			cobs.WithSentinel(byte(encFlags.sentinel)),
			cobs.WithDelimiterOnClose(*appendDelim),
			cobs.WithReduced(encFlags.reduced))

		if _, err := io.Copy(enc, os.Stdin); err != nil {
			panic(err)
		}

		if err := enc.Close(); err != nil {
			panic(err)
		}

	case "decode":
		decodeCmd.Parse(os.Args[2:])

		if err := decFlags.validate(); err != nil {
			fatal(err)
		}

		dec := cobs.NewDecoder(
			os.Stdout,
			cobs.WithSentinel(byte(decFlags.sentinel)),
			cobs.WithReduced(decFlags.reduced))

		if _, err := io.Copy(dec, os.Stdin); err != nil && err != cobs.EOD {
			panic(err)
		}

		if err := dec.Close(); err != nil {
			panic(err)
		}

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\nUsage: cobs <encode|decode> [flags]\n", os.Args[1])
		os.Exit(1)
	}
}
