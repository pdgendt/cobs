package cobs

import (
	"bufio"
	"bytes"
	"io"
	"sync"
	"testing"
)

var testCases = []struct {
	name    string
	dec, enc []byte
}{
	{
		name: "Empty",
		dec:   []byte{},
		enc:   []byte{0x01},
	},
	{
		name: "1 character",
		dec:   []byte{'1'},
		enc:   []byte{0x02, '1'},
	},
	{
		name: "1 zero",
		dec:   []byte{0x00},
		enc:   []byte{0x01, 0x01},
	},
	{
		name: "2 zeroes",
		dec:   []byte{0x00, 0x00},
		enc:   []byte{0x01, 0x01, 0x01},
	},
	{
		name: "3 zeroes",
		dec:   []byte{0x00, 0x00, 0x00},
		enc:   []byte{0x01, 0x01, 0x01, 0x01},
	},
	{
		name: "5 characters",
		dec:   []byte("12345"),
		enc:   []byte("\x0612345"),
	},
	{
		name: "Embedded zero",
		dec:   []byte("12345\x006789"),
		enc:   []byte("\x0612345\x056789"),
	},
	{
		name: "Starting and embedded zero",
		dec:   []byte("\x0012345\x006789"),
		enc:   []byte("\x01\x0612345\x056789"),
	},
	{
		name: "Embedded and trailing zero",
		dec:   []byte("12345\x006789\x00"),
		enc:   []byte("\x0612345\x056789\x01"),
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
			e := NewEncoder(NewDecoder(buf))

			n, err := e.Write(tc.dec)
			if err != nil {
				t.Errorf("writer error: %v", err)
			}
			err = e.Close()
			if err != nil {
				t.Errorf("writer close error: %v", err)
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
	var err error
	var wg sync.WaitGroup

	pr, pw := io.Pipe()
	defer pw.Close()

	go func() {
		r := bufio.NewReader(pr)

		var buf bytes.Buffer
		d := NewDecoder(&buf)

		for i := 0; ; i++ {
			// Read until delimiter
			tmp, err := r.ReadBytes(Delimiter)
			if err != nil {
				if err != io.EOF {
					t.Errorf("pipe error: %v", err)
				}
				return
			}

			_, err = d.Write(tmp)
			if err != EOD {
				t.Error("stream decode EOD missing")
			}
			if !bytes.Equal(buf.Bytes(), testCases[i].dec) {
				t.Errorf("stream decode got %v, want %v", buf.Bytes(), testCases[i].dec)
			}

			buf.Reset()

			t.Logf("stream decode %s", testCases[i].name)
			wg.Done()
		}

	}()

	e := NewEncoder(pw)
	for _, tc := range testCases {
		wg.Add(1)

		_, err = e.Write(tc.dec)
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

	wg.Wait()
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
		e := NewEncoder(NewDecoder(&buf))

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
		if !bytes.Equal(buf.Bytes(), a) {
			t.Errorf("fuzz chain got %v want %v", buf.Bytes(), a)
		}
	})
}
