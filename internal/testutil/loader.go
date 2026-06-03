// Package testutil provides shared test helpers for the cobs module.
package testutil

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// Case is a single encode/decode fixture pair loaded from disk.
type Case struct {
	Name     string
	Reduced  bool
	Sentinel byte
	Encoded  []byte
	Decoded  []byte
}

// fixtureRE matches fixture filenames of the form {name}_{XX}[_r].out,
// where XX is the sentinel as two hex digits and _r marks COBS/R.
var fixtureRE = regexp.MustCompile(`^(.+)_([0-9a-f]{2})(_r)?\.out$`)

// Load reads all fixture pairs from dir. For each {name}_{XX}[_r].out file
// it pairs the encoded bytes with the matching {name}.in file. The test
// is failed if dir is unreadable, a filename is malformed, or a pair is
// incomplete.
func Load(tb testing.TB, dir string) []Case {
	tb.Helper()

	files, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("Failed to read testdata directory: %v", err)
	}

	var cases []Case

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".out") {
			continue
		}

		matches := fixtureRE.FindStringSubmatch(filename)
		if matches == nil {
			tb.Logf("Skipping file with invalid name format: %s", filename)
			continue
		}

		name := matches[1]
		sentinelVal, err := strconv.ParseUint(matches[2], 16, 8)
		if err != nil {
			tb.Fatalf("Failed to parse sentinel hex %q in %s: %v", matches[2], filename, err)
		}

		enc, err := os.ReadFile(filepath.Join(dir, filename))
		if err != nil {
			tb.Fatalf("Failed to read %s: %v", filename, err)
		}

		dec, err := os.ReadFile(filepath.Join(dir, name+".in"))
		if err != nil {
			tb.Fatalf("Failed to read %s.in: %v", name, err)
		}

		cases = append(cases, Case{
			Name:     name,
			Reduced:  matches[3] == "_r",
			Sentinel: byte(sentinelVal),
			Encoded:  enc,
			Decoded:  dec,
		})
	}

	if len(cases) == 0 {
		tb.Fatalf("No test cases found in %s", dir)
	}

	return cases
}
