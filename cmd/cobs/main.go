/*
Cobs reads from standard input, and writes encoded or decoded data to standard output.

Usage:

	cobs <command> [flags]
	cobs -h/--help
	cobs -V/--version

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
is passed the program will exit with an error.
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/pdgendt/cobs"
)

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: cobs <command> [flags]")
	fmt.Fprintln(w, "       cobs -h/--help")
	fmt.Fprintln(w, "       cobs -V/--version")
	fmt.Fprintln(w, "\nCommands:")
	fmt.Fprintln(w, "  encode    Encode data using COBS")
	fmt.Fprintln(w, "  decode    Decode COBS-encoded data")
	fmt.Fprintln(w, "\nRun 'cobs <command> -h' for command-specific flags.")
}

func version() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "(devel)"
}

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
		return errors.New("sentinel value must be in [0x00, 0xFF]")
	}

	return nil
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	var encFlags, decFlags commonFlags

	encodeCmd := flag.NewFlagSet("encode", flag.ContinueOnError)
	encodeCmd.SetOutput(stderr)
	encFlags.register(encodeCmd)
	appendDelim := encodeCmd.Bool("del", false, "Append a delimiter")
	encodeCmd.BoolVar(appendDelim, "d", false, "Append a delimiter")

	decodeCmd := flag.NewFlagSet("decode", flag.ContinueOnError)
	decodeCmd.SetOutput(stderr)
	decFlags.register(decodeCmd)

	if len(args) == 0 {
		usage(stderr)
		return 1
	}

	switch args[0] {
	case "-h", "--help":
		usage(stderr)
		return 0

	case "-V", "--version":
		fmt.Fprintln(stdout, version())
		return 0

	case "encode":
		if err := encodeCmd.Parse(args[1:]); err != nil {
			return 1
		}
		if err := encFlags.validate(); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		enc := cobs.NewEncoder(
			stdout,
			cobs.WithSentinel(byte(encFlags.sentinel)),
			cobs.WithDelimiterOnClose(*appendDelim),
			cobs.WithReduced(encFlags.reduced))

		if _, err := io.Copy(enc, stdin); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		if err := enc.Close(); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		return 0

	case "decode":
		if err := decodeCmd.Parse(args[1:]); err != nil {
			return 1
		}
		if err := decFlags.validate(); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		dec := cobs.NewDecoder(
			stdout,
			cobs.WithSentinel(byte(decFlags.sentinel)),
			cobs.WithReduced(decFlags.reduced))

		if _, err := io.Copy(dec, stdin); err != nil && err != cobs.EOD {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		if err := dec.Close(); err != nil {
			fmt.Fprintf(stderr, "Error: %v\n", err)
			return 1
		}

		return 0

	default:
		fmt.Fprintf(stderr, "Unknown command: %s\n", args[0])
		usage(stderr)
		return 1
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
