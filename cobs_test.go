package cobs

import (
	"bytes"
	"io"
	"testing"
)

var testCases = []struct {
	name     string
	dec, enc []byte
}{
	{
		name: "Empty",
		dec:  []byte{},
		enc:  []byte{0x01},
	},
	{
		name: "1 character",
		dec:  []byte{'1'},
		enc:  []byte{0x02, '1'},
	},
	{
		name: "1 zero",
		dec:  []byte{0x00},
		enc:  []byte{0x01, 0x01},
	},
	{
		name: "2 zeroes",
		dec:  []byte{0x00, 0x00},
		enc:  []byte{0x01, 0x01, 0x01},
	},
	{
		name: "3 zeroes",
		dec:  []byte{0x00, 0x00, 0x00},
		enc:  []byte{0x01, 0x01, 0x01, 0x01},
	},
	{
		name: "5 characters",
		dec:  []byte("12345"),
		enc:  []byte("\x0612345"),
	},
	{
		name: "Embedded zero",
		dec:  []byte("12345\x006789"),
		enc:  []byte("\x0612345\x056789"),
	},
	{
		name: "Starting and embedded zero",
		dec:  []byte("\x0012345\x006789"),
		enc:  []byte("\x01\x0612345\x056789"),
	},
	{
		name: "Embedded and trailing zero",
		dec:  []byte("12345\x006789\x00"),
		enc:  []byte("\x0612345\x056789\x01"),
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
	},
}

func TestEncode(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			enc, err := Encode(tc.dec)
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
		t.Run(tc.name, func(t *testing.T) {
			dec, err := Decode(tc.enc)
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
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, len(tc.enc)))
			d := NewDecoder(buf)
			e := NewEncoder(d)

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

func TestStream(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		defer pw.Close()
		e := NewEncoder(pw)

		for _, tc := range testCases {
			_, err := e.Write(tc.dec)
			if err != nil {
				t.Errorf("stream encode error: %v", err)
			}
			err = e.Close()
			if err != nil {
				t.Errorf("stream close error: %v", err)
			}

			_, err = pw.Write([]byte{Delimiter})
			if err != nil {
				t.Errorf("stream delimiter error: %v", err)
			}
		}
	}()

	var buf bytes.Buffer
	d := NewDecoder(&buf)

	for _, tc := range testCases {
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
		f.Add(tc.dec)
	}
	f.Fuzz(func(t *testing.T, a []byte) {
		enc, err := Encode(a)
		if err != nil {
			t.Errorf("fuzz encode error: %v", err)
		}
		if i := bytes.IndexByte(enc, Delimiter); i != -1 {
			t.Errorf("fuzz encode %v has delimiter at %d", enc, i)
		}

		dec, err := Decode(enc)
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
		f.Add(tc.dec)
	}
	f.Fuzz(func(t *testing.T, a []byte) {
		var buf bytes.Buffer
		d := NewDecoder(&buf)
		e := NewEncoder(d)

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
		t.Run(tc.name, func(t *testing.T) {
			e := NewEncoder(&LimitedWriter{io.Discard, 0, io.EOF})

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
		t.Run(tc.name, func(t *testing.T) {
			// The empty string is expected to have no writes
			if len(tc.dec) == 0 {
				t.SkipNow()
			}

			d := NewDecoder(&LimitedWriter{io.Discard, 0, io.EOF})

			_, err := d.Write(tc.enc)
			if err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
