package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestRunUsage(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantCode int
		wantErr  string
	}{
		{"no args", nil, 1, "Usage:"},
		{"help -h", []string{"-h"}, 0, "Usage:"},
		{"help --help", []string{"--help"}, 0, "Usage:"},
		{"unknown command", []string{"bogus"}, 1, "Unknown command: bogus"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(""), &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if !strings.Contains(stderr.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), tc.wantErr)
			}
		})
	}
}

func TestRunVersion(t *testing.T) {
	for _, flag := range []string{"-V", "--version"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run([]string{flag}, strings.NewReader(""), &stdout, &stderr)
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
			if strings.TrimSpace(stdout.String()) == "" {
				t.Error("expected version output, got empty")
			}
		})
	}
}

func TestRunFlags(t *testing.T) {
	input := []byte("Hello, World!")

	t.Run("custom sentinel", func(t *testing.T) {
		var encOut, encErr bytes.Buffer
		code := run([]string{"encode", "-s", "127"}, bytes.NewReader(input), &encOut, &encErr)
		if code != 0 {
			t.Fatalf("encode exit %d: %s", code, encErr.String())
		}

		var decOut, decErr bytes.Buffer
		code = run([]string{"decode", "-s", "127"}, bytes.NewReader(encOut.Bytes()), &decOut, &decErr)
		if code != 0 {
			t.Fatalf("decode exit %d: %s", code, decErr.String())
		}

		if !bytes.Equal(decOut.Bytes(), input) {
			t.Errorf("round-trip failed: got %v, want %v", decOut.Bytes(), input)
		}
	})

	t.Run("reduced mode", func(t *testing.T) {
		var encOut, encErr bytes.Buffer
		code := run([]string{"encode", "-r"}, bytes.NewReader(input), &encOut, &encErr)
		if code != 0 {
			t.Fatalf("encode exit %d: %s", code, encErr.String())
		}

		var decOut, decErr bytes.Buffer
		code = run([]string{"decode", "-r"}, bytes.NewReader(encOut.Bytes()), &decOut, &decErr)
		if code != 0 {
			t.Fatalf("decode exit %d: %s", code, decErr.String())
		}

		if !bytes.Equal(decOut.Bytes(), input) {
			t.Errorf("round-trip failed: got %v, want %v", decOut.Bytes(), input)
		}
	})

	t.Run("delimiter flag", func(t *testing.T) {
		var encOut, encErr bytes.Buffer
		code := run([]string{"encode", "-d"}, bytes.NewReader(input), &encOut, &encErr)
		if code != 0 {
			t.Fatalf("encode exit %d: %s", code, encErr.String())
		}

		encoded := encOut.Bytes()
		if len(encoded) == 0 {
			t.Fatal("encoded output is empty")
		}
		if encoded[len(encoded)-1] != 0x00 {
			t.Errorf("expected trailing delimiter 0x00, got 0x%02x", encoded[len(encoded)-1])
		}
	})

	t.Run("reduced with custom sentinel", func(t *testing.T) {
		var encOut, encErr bytes.Buffer
		code := run([]string{"encode", "-r", "-s", "255"}, bytes.NewReader(input), &encOut, &encErr)
		if code != 0 {
			t.Fatalf("encode exit %d: %s", code, encErr.String())
		}

		var decOut, decErr bytes.Buffer
		code = run([]string{"decode", "-r", "-s", "255"}, bytes.NewReader(encOut.Bytes()), &decOut, &decErr)
		if code != 0 {
			t.Fatalf("decode exit %d: %s", code, decErr.String())
		}

		if !bytes.Equal(decOut.Bytes(), input) {
			t.Errorf("round-trip failed: got %v, want %v", decOut.Bytes(), input)
		}
	})
}

func TestRunErrors(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		input    string
		wantCode int
		wantErr  string
	}{
		{"invalid sentinel encode", []string{"encode", "-s", "256"}, "", 1, "sentinel value must be in"},
		{"invalid sentinel decode", []string{"decode", "-s", "256"}, "", 1, "sentinel value must be in"},
		{"negative sentinel", []string{"encode", "-s", "-1"}, "", 1, "sentinel value must be in"},
		{"invalid flag encode", []string{"encode", "-z"}, "", 1, ""},
		{"invalid flag decode", []string{"decode", "-z"}, "", 1, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tc.args, strings.NewReader(tc.input), &stdout, &stderr)
			if code != tc.wantCode {
				t.Errorf("exit code = %d, want %d", code, tc.wantCode)
			}
			if tc.wantErr != "" && !strings.Contains(stderr.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), tc.wantErr)
			}
		})
	}
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

func TestRunIOErrors(t *testing.T) {
	ioErr := fmt.Errorf("injected I/O error")

	tests := []struct {
		name    string
		args    []string
		stdin   *errReader
		stdout  *errWriter
		wantErr string
	}{
		{"encode read error", []string{"encode"}, &errReader{ioErr}, nil, "injected I/O error"},
		{"encode write error", []string{"encode"}, nil, &errWriter{ioErr}, "injected I/O error"},
		{"decode read error", []string{"decode"}, &errReader{ioErr}, nil, "injected I/O error"},
		{"decode write error", []string{"decode"}, nil, &errWriter{ioErr}, "injected I/O error"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdin bytes.Reader
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			var r interface{ Read([]byte) (int, error) } = &stdin
			var w interface{ Write([]byte) (int, error) } = &stdout

			if tc.stdin != nil {
				r = tc.stdin
			} else {
				// Provide valid input that will trigger a write
				stdin = *bytes.NewReader([]byte("test data"))
			}
			if tc.stdout != nil {
				w = tc.stdout
			}

			code := run(tc.args, r, w, &stderr)
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if !strings.Contains(stderr.String(), tc.wantErr) {
				t.Errorf("stderr = %q, want substring %q", stderr.String(), tc.wantErr)
			}
		})
	}
}

type limitWriter struct {
	w         *bytes.Buffer
	remaining int
	err       error
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.remaining <= 0 {
		return 0, w.err
	}
	if len(p) > w.remaining {
		n, _ := w.w.Write(p[:w.remaining])
		w.remaining = 0
		return n, w.err
	}
	n, err := w.w.Write(p)
	w.remaining -= n
	return n, err
}

func TestRunCloseErrors(t *testing.T) {
	ioErr := fmt.Errorf("injected write error")

	t.Run("encode close error", func(t *testing.T) {
		// Short input with no zeroes: encoder buffers everything, writes only on Close
		var stderr bytes.Buffer
		w := &limitWriter{w: &bytes.Buffer{}, remaining: 0, err: ioErr}
		code := run([]string{"encode"}, bytes.NewReader([]byte("test")), w, &stderr)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
		if !strings.Contains(stderr.String(), "injected write error") {
			t.Errorf("stderr = %q, want injected write error", stderr.String())
		}
	})

	t.Run("decode close error", func(t *testing.T) {
		// With COBS/R, the decoder's Close flushes the last byte via flushReduced.
		// Decode "test" (4 bytes): decoder writes 3 bytes during io.Copy,
		// flushReduced writes the 4th on Close.
		// Allow exactly the io.Copy writes to succeed, then fail on Close's write.
		var encOut, encErr bytes.Buffer
		run([]string{"encode", "-r"}, bytes.NewReader([]byte("test")), &encOut, &encErr)

		// Decode into a good buffer first to count how many bytes io.Copy writes
		var countBuf bytes.Buffer
		var countErr bytes.Buffer
		run([]string{"decode", "-r"}, bytes.NewReader(encOut.Bytes()), &countBuf, &countErr)
		// flushReduced writes 1 byte on Close; allow all but that last byte
		decBytes := countBuf.Len() - 1

		var stderr bytes.Buffer
		w := &limitWriter{w: &bytes.Buffer{}, remaining: decBytes, err: ioErr}
		code := run([]string{"decode", "-r"}, bytes.NewReader(encOut.Bytes()), w, &stderr)
		if code != 1 {
			t.Errorf("exit code = %d, want 1", code)
		}
	})
}

func TestRunTestdata(t *testing.T) {
	testdataDir := filepath.Join("..", "..", "testdata")

	re := regexp.MustCompile(`^(.+)_([0-9a-f]{2})(_r)?\.out$`)

	files, err := os.ReadDir(testdataDir)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".out") {
			continue
		}

		matches := re.FindStringSubmatch(file.Name())
		if matches == nil {
			continue
		}

		name := matches[1]
		sentinelHex := matches[2]
		reduced := matches[3] == "_r"

		sentinel, err := strconv.ParseUint(sentinelHex, 16, 8)
		if err != nil {
			t.Fatalf("Failed to parse sentinel hex %q: %v", sentinelHex, err)
		}

		inFile := filepath.Join(testdataDir, name+".in")
		outFile := filepath.Join(testdataDir, file.Name())

		inData, err := os.ReadFile(inFile)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", inFile, err)
		}

		wantEncoded, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", outFile, err)
		}

		testName := fmt.Sprintf("%s (sentinel=0x%s, reduced=%v)", name, sentinelHex, reduced)

		t.Run(testName+"/encode", func(t *testing.T) {
			args := []string{"encode", "-s", fmt.Sprintf("%d", sentinel)}
			if reduced {
				args = append(args, "-r")
			}

			var stdout, stderr bytes.Buffer
			code := run(args, bytes.NewReader(inData), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("encode exit %d: %s", code, stderr.String())
			}

			if !bytes.Equal(stdout.Bytes(), wantEncoded) {
				t.Errorf("encode mismatch:\n  got  %v\n  want %v", stdout.Bytes(), wantEncoded)
			}
		})

		t.Run(testName+"/decode", func(t *testing.T) {
			args := []string{"decode", "-s", fmt.Sprintf("%d", sentinel)}
			if reduced {
				args = append(args, "-r")
			}

			var stdout, stderr bytes.Buffer
			code := run(args, bytes.NewReader(wantEncoded), &stdout, &stderr)
			if code != 0 {
				t.Fatalf("decode exit %d: %s", code, stderr.String())
			}

			if !bytes.Equal(stdout.Bytes(), inData) {
				t.Errorf("decode mismatch:\n  got  %v\n  want %v", stdout.Bytes(), inData)
			}
		})
	}
}
