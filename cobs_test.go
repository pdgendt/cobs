package cobs

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

var testCases = []struct {
	name     string
	dec, enc []byte
	sentinel byte
}{
	{
		name:     "Empty",
		dec:      []byte{},
		enc:      []byte{0x01},
		sentinel: Delimiter,
	},
	{
		name:     "Empty",
		dec:      []byte{},
		enc:      []byte{0x00},
		sentinel: 0x01,
	},
	{
		name:     "Empty",
		dec:      []byte{},
		enc:      []byte{0x00},
		sentinel: 0xFF,
	},
	{
		name:     "1 character",
		dec:      []byte{'1'},
		enc:      []byte{0x02, '1'},
		sentinel: Delimiter,
	},
	{
		name:     "1 character",
		dec:      []byte{'1'},
		enc:      []byte{0x01, 0x31},
		sentinel: 0xFF,
	},
	{
		name:     "1 character",
		dec:      []byte{'A'},
		enc:      []byte{0x01, 0x41},
		sentinel: 0x7F,
	},
	{
		name:     "1 zero",
		dec:      []byte{0x00},
		enc:      []byte{0x01, 0x01},
		sentinel: Delimiter,
	},
	{
		name:     "1 sentinel byte (0xFF)",
		dec:      []byte{0xFF},
		enc:      []byte{0x00, 0x00},
		sentinel: 0xFF,
	},
	{
		name:     "1 sentinel byte (0x01)",
		dec:      []byte{0x01},
		enc:      []byte{0x00, 0x00},
		sentinel: 0x01,
	},
	{
		name:     "2 zeroes",
		dec:      []byte{0x00, 0x00},
		enc:      []byte{0x01, 0x01, 0x01},
		sentinel: Delimiter,
	},
	{
		name:     "2 sentinel bytes (0xFF)",
		dec:      []byte{0xFF, 0xFF},
		enc:      []byte{0x00, 0x00, 0x00},
		sentinel: 0xFF,
	},
	{
		name:     "3 zeroes",
		dec:      []byte{0x00, 0x00, 0x00},
		enc:      []byte{0x01, 0x01, 0x01, 0x01},
		sentinel: Delimiter,
	},
	{
		name:     "5 characters",
		dec:      []byte("12345"),
		enc:      []byte("\x0612345"),
		sentinel: Delimiter,
	},
	{
		name:     "5 characters",
		dec:      []byte{'1', '2', '3', '4', '5'},
		enc:      []byte{0x05, 0x31, 0x32, 0x33, 0x34, 0x35},
		sentinel: 0xFF,
	},
	{
		name:     "Embedded zero",
		dec:      []byte("12345\x006789"),
		enc:      []byte("\x0612345\x056789"),
		sentinel: Delimiter,
	},
	{
		name:     "Embedded sentinel (0xFF)",
		dec:      []byte{'1', '2', '3', '4', '5', 0xFF, '6', '7', '8', '9'},
		enc:      []byte{0x05, 0x31, 0x32, 0x33, 0x34, 0x35, 0x04, 0x36, 0x37, 0x38, 0x39},
		sentinel: 0xFF,
	},
	{
		name:     "Multiple embedded 0xFF",
		dec:      []byte{'A', 'B', 0xFF, 'C', 'D', 0xFF, 'E', 'F'},
		enc:      []byte{0x02, 0x41, 0x42, 0x02, 0x43, 0x44, 0x02, 0x45, 0x46},
		sentinel: 0xFF,
	},
	{
		name:     "Starting and embedded zero",
		dec:      []byte("\x0012345\x006789"),
		enc:      []byte("\x01\x0612345\x056789"),
		sentinel: Delimiter,
	},
	{
		name: "Starting and embedded sentinel (0xFF)",
		dec:  []byte{0xFF, '1', '2', '3', '4', '5', 0xFF, '6', '7', '8', '9'},
		enc: []byte{0x00, 0x05, 0x31, 0x32, 0x33, 0x34, 0x35, 0x04, 0x36, 0x37, 0x38,
			0x39},
		sentinel: 0xFF,
	},
	{
		name:     "Embedded and trailing zero",
		dec:      []byte("12345\x006789\x00"),
		enc:      []byte("\x0612345\x056789\x01"),
		sentinel: Delimiter,
	},
	{
		name: "Embedded and trailing sentinel (0xFF)",
		dec:  []byte{'1', '2', '3', '4', '5', 0xFF, '6', '7', '8', '9', 0xFF},
		enc: []byte{0x05, 0x31, 0x32, 0x33, 0x34, 0x35, 0x04, 0x36, 0x37, 0x38, 0x39,
			0x00},
		sentinel: 0xFF,
	},
	{
		name:     "Starting, embedded and trailing sentinel (0xFF)",
		dec:      []byte{0xFF, '1', '2', 0xFF, '3', '4', 0xFF},
		enc:      []byte{0x00, 0x02, 0x31, 0x32, 0x02, 0x33, 0x34, 0x00},
		sentinel: 0xFF,
	},
	{
		name: "253 non-zero bytes",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst123"),
		enc: []byte("\xfe0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst123"),
		sentinel: Delimiter,
	},
	{
		name: "254 non-zero bytes",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234"),
		enc: []byte("\xff0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234"),
		sentinel: Delimiter,
	},
	{
		name: "255 non-zero bytes",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst12345"),
		enc: []byte("\xff0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234\x025"),
		sentinel: Delimiter,
	},
	{
		name: "zero followed by 255 non-zero bytes",
		dec: []byte("\x000123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst12345"),
		enc: []byte("\x01\xff0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01234567" +
			"89ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQR" +
			"STabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqr" +
			"st0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234\x025"),
		sentinel: Delimiter,
	},
	{
		name: "253 non-zero bytes followed by zero",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst123\x00"),
		enc: []byte("\xfe0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst123\x01"),
		sentinel: Delimiter,
	},
	{
		name: "254 non-zero bytes followed by zero",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234\x00"),
		enc: []byte("\xff0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234\x01\x01"),
		sentinel: Delimiter,
	},
	{
		name: "255 non-zero bytes followed by zero",
		dec: []byte("0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEF" +
			"GHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdef" +
			"ghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst012345" +
			"6789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst12345\x00"),
		enc: []byte("\xff0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789AB" +
			"CDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTab" +
			"cdefghijklmnopqrst0123456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst01" +
			"23456789ABCDEFGHIJKLMNOPQRSTabcdefghijklmnopqrst1234\x025\x01"),
		sentinel: Delimiter,
	},
	{
		name:     "All 0xFF bytes",
		dec:      []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
		enc:      []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		sentinel: 0xFF,
	},
	{
		name:     "Alternating 0xFF and non-0xFF",
		dec:      []byte{0xFF, 'A', 0xFF, 'B', 0xFF},
		enc:      []byte{0x00, 0x01, 0x41, 0x01, 0x42, 0x00},
		sentinel: 0xFF,
	},
}

func TestEncode(t *testing.T) {
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

	go func() {
		defer pw.Close()

		for _, tc := range testCases {
			e := NewEncoder(pw, WithSentinel(tc.sentinel))
			_, err := e.Write(tc.dec)
			if err != nil {
				t.Errorf("stream encode error: %v", err)
			}
			err = e.Close()
			if err != nil {
				t.Errorf("stream close error: %v", err)
			}

			_, err = pw.Write([]byte{tc.sentinel})
			if err != nil {
				t.Errorf("stream delimiter error: %v", err)
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
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.name, tc.sentinel), func(t *testing.T) {
			e := NewEncoder(
				&LimitedWriter{io.Discard, 0, io.EOF},
				WithSentinel(tc.sentinel))

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
