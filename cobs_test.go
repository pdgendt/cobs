package cobs

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

type testCase struct {
	name     string
	enc      []byte
	dec      []byte
	sentinel byte
	reduced  bool
}

// loadTestCasesFromFiles loads test cases from .out and matching .in file pairs in the specified
// directory.
// Filename format: {name}_{XX}.out where XX is hex sentinel.
// For reduced mode: {name}_{XX}_r.out
func loadTestCasesFromFiles(tb testing.TB, dir string) []testCase {
	tb.Helper()

	// Pattern to match filenames: name-XX[_r].out
	// Captures: name (group 1), sentinel hex (group 2), _r if present (group 3)
	re := regexp.MustCompile(`^(.+)_([0-9a-f]{2})(_r)?\.out$`)

	files, err := os.ReadDir(dir)
	if err != nil {
		tb.Fatalf("Failed to read testdata directory: %v", err)
	}

	var testCases []testCase

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := file.Name()
		if !strings.HasSuffix(filename, ".out") {
			continue
		}

		matches := re.FindStringSubmatch(filename)
		if matches == nil {
			tb.Logf("Skipping file with invalid name format: %s", filename)
			continue
		}

		// Extract components from filename
		name := matches[1]
		sentinelHex := matches[2]
		reduced := matches[3] == "_r"

		// Parse sentinel value from hex
		sentinelVal, err := strconv.ParseUint(sentinelHex, 16, 8)
		if err != nil {
			tb.Fatalf("Failed to parse sentinel hex '%s' in %s: %v", sentinelHex, filename, err)
		}
		sentinel := byte(sentinelVal)

		// Build display name
		displayName := name
		if reduced {
			displayName += " [reduced]"
		}

		// Read output file (encoded bytes)
		outPath := filepath.Join(dir, filename)
		enc, err := os.ReadFile(outPath)
		if err != nil {
			tb.Fatalf("Failed to read %s: %v", outPath, err)
		}

		// Read input file (decoded bytes)
		inFilename := name + ".in"
		inPath := filepath.Join(dir, inFilename)
		dec, err := os.ReadFile(inPath)
		if err != nil {
			tb.Fatalf("Failed to read %s: %v", inPath, err)
		}

		testCases = append(testCases, testCase{
			name:     displayName,
			dec:      dec,
			enc:      enc,
			sentinel: sentinel,
			reduced:  reduced,
		})
	}

	if len(testCases) == 0 {
		tb.Fatalf("No test cases found in %s", dir)
	}

	return testCases
}

func TestEncode(t *testing.T) {
	testCases := loadTestCasesFromFiles(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.name, tc.sentinel), func(t *testing.T) {
			enc, err := Encode(tc.dec, WithSentinel(tc.sentinel))
			if err != nil {
				t.Errorf("encode error: %v", err)
			}
			if !bytes.Equal(enc, tc.enc) {
				t.Errorf("got %v, want %v", enc, tc.enc)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := loadTestCasesFromFiles(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.name, tc.sentinel), func(t *testing.T) {
			dec, err := Decode(tc.enc, WithSentinel(tc.sentinel))
			if err != nil {
				t.Errorf("decode error: %v", err)
			}
			if !bytes.Equal(dec, tc.dec) {
				t.Errorf("got %v, want %v", dec, tc.dec)
			}
		})
	}
}

func TestWriter(t *testing.T) {
	testCases := loadTestCasesFromFiles(t, "testdata")
	// This test has to work with all sentinel values (encoded buffer is not compared)
	for _, tc := range testCases {
		for s := 0; s <= 255; s++ {
			t.Run(fmt.Sprintf("%s (%d)", tc.name, s), func(t *testing.T) {
				buf := bytes.NewBuffer(make([]byte, 0, len(tc.enc)))
				d := NewDecoder(buf, WithSentinel(byte(s)))
				e := NewEncoder(d, WithSentinel(byte(s)))

				n, err := e.Write(tc.dec)
				if err != nil {
					t.Errorf("writer error: %v", err)
				}
				err = e.Close()
				if err != nil {
					t.Errorf("writer close error: %v", err)
				}
				if d.NeedsMoreData() {
					t.Error("writer incomplete decode data")
				}
				if n != len(tc.dec) {
					t.Errorf("writer length got %d, want %d", n, len(tc.dec))
				}
				if !bytes.Equal(buf.Bytes(), tc.dec) {
					t.Errorf("got %v, want %v", buf.Bytes(), tc.dec)
				}
			})
		}
	}
}

func TestStream(t *testing.T) {
	pr, pw := io.Pipe()
	testCases := loadTestCasesFromFiles(t, "testdata")

	go func() {
		defer pw.Close()

		for _, tc := range testCases {
			e := NewEncoder(pw, WithSentinel(tc.sentinel), WithDelimiterOnClose(true))
			_, err := e.Write(tc.dec)
			if err != nil {
				t.Errorf("stream encode error: %v", err)
			}
			err = e.Close()
			if err != nil {
				t.Errorf("stream close error: %v", err)
			}
		}
	}()

	var buf bytes.Buffer

	for _, tc := range testCases {
		d := NewDecoder(&buf, WithSentinel(tc.sentinel))
		_, err := io.Copy(d, pr)

		if err != EOD {
			t.Error("stream decode EOD missing")
		}

		if d.NeedsMoreData() {
			t.Error("stream decode frame incomplete")
		}

		if !bytes.Equal(buf.Bytes(), tc.dec) {
			t.Errorf("stream decode got %v, want %v", buf.Bytes(), tc.dec)
		}

		buf.Reset()

		t.Logf("stream decode %s", tc.name)
	}
}

func FuzzEncodeDecode(f *testing.F) {
	testCases := loadTestCasesFromFiles(f, "testdata")
	for _, tc := range testCases {
		for s := 0; s <= 255; s++ {
			f.Add(tc.dec, byte(s))
		}
	}
	f.Fuzz(func(t *testing.T, a []byte, del byte) {
		enc, err := Encode(a, WithSentinel(del))
		if err != nil {
			t.Errorf("fuzz encode error: %v", err)
		}
		if i := bytes.IndexByte(enc, del); i != -1 {
			t.Errorf("fuzz encode %v has sentinel at %d", enc, i)
		}

		dec, err := Decode(enc, WithSentinel(del))
		if err != nil {
			t.Errorf("fuzz decode error: %v", err)
		}
		if !bytes.Equal(dec, a) {
			t.Errorf("fuzz decode got %v want %v", dec, a)
		}
	})
}

func FuzzChainWriter(f *testing.F) {
	testCases := loadTestCasesFromFiles(f, "testdata")
	for _, tc := range testCases {
		for del := 0; del <= 255; del++ {
			f.Add(tc.dec, byte(del))
		}
	}
	f.Fuzz(func(t *testing.T, a []byte, del byte) {
		var buf bytes.Buffer
		d := NewDecoder(&buf, WithSentinel(del))
		e := NewEncoder(d, WithSentinel(del))

		n, err := e.Write(a)
		if err != nil {
			t.Errorf("fuzz chain error: %v", err)
		}
		if n != len(a) {
			t.Errorf("fuzz chain length got %d want %d", n, len(a))
		}

		err = e.Close()
		if err != nil {
			t.Errorf("fuzz chain close error: %v", err)
		}
		if d.NeedsMoreData() {
			t.Error("fuzz chain incomplete decode data")
		}
		if !bytes.Equal(buf.Bytes(), a) {
			t.Errorf("fuzz chain got %v want %v", buf.Bytes(), a)
		}
	})
}

var incompleteFrame = []struct {
	name string
	data []byte
}{
	{
		name: "Missing single byte",
		data: []byte{0x02},
	},
	{
		name: "2 zeroes and missing end",
		data: []byte{0x01, 0x01, 0x05},
	},
}

func TestDecodeIncomplete(t *testing.T) {
	for _, tc := range incompleteFrame {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.data)
			if err != ErrIncompleteFrame {
				t.Errorf("Unexpected decode incomplete error: %v", err)
			}
		})
	}
}

var unexpectedDelimiter = []struct {
	name string
	data []byte
}{
	{
		name: "Missing byte before delimiter",
		data: []byte{0x02, 0x00},
	},
	{
		name: "Unexpected embedded zero",
		data: []byte("\x061234\x005\x056789"),
	},
}

func TestDecodeUnexpectedEOD(t *testing.T) {
	for _, tc := range unexpectedDelimiter {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Decode(tc.data)
			if err != ErrUnexpectedEOD {
				t.Errorf("Unexpected decode EOD error: %v", err)
			}
		})
	}
}

// https://github.com/golang/go/issues/54111
type LimitedWriter struct {
	W   io.Writer // underlying writer
	N   int64     // max bytes remaining
	Err error     // error to be returned once limit is reached
}

func (lw *LimitedWriter) Write(p []byte) (int, error) {
	if lw.N < 1 {
		return 0, lw.Err
	}
	if lw.N < int64(len(p)) {
		p = p[:lw.N]
	}
	n, err := lw.W.Write(p)
	lw.N -= int64(n)
	return n, err
}

func TestEncodeError(t *testing.T) {
	testCases := loadTestCasesFromFiles(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.name, tc.sentinel), func(t *testing.T) {
			e := NewEncoder(
				&LimitedWriter{io.Discard, 1, io.EOF},
				WithSentinel(tc.sentinel),
				WithDelimiterOnClose(true))

			_, err := e.Write(tc.dec)
			// err can be nil if no groups have been flushed, call close
			if err == nil {
				err = e.Close()
			}
			if err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDecodeError(t *testing.T) {
	testCases := loadTestCasesFromFiles(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.name, tc.sentinel), func(t *testing.T) {
			// The empty string is expected to have no writes
			if len(tc.dec) == 0 {
				t.SkipNow()
			}

			d := NewDecoder(
				&LimitedWriter{io.Discard, 0, io.EOF},
				WithSentinel(tc.sentinel))

			_, err := d.Write(tc.enc)
			if err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
