package cobs

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/pdgendt/cobs/internal/testutil"
)

func displayName(tc testutil.Case) string {
	if tc.Reduced {
		return tc.Name + " [reduced]"
	}
	return tc.Name
}

func TestEncode(t *testing.T) {
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			enc, err := Encode(tc.Decoded,
				WithSentinel(tc.Sentinel),
				WithReduced(tc.Reduced))
			if err != nil {
				t.Errorf("encode error: %v", err)
			}
			if !bytes.Equal(enc, tc.Encoded) {
				t.Errorf("got %v, want %v", enc, tc.Encoded)
			}
		})
	}
}

func TestDecode(t *testing.T) {
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			dec, err := Decode(tc.Encoded,
				WithSentinel(tc.Sentinel),
				WithReduced(tc.Reduced))
			if err != nil {
				t.Errorf("decode error: %v", err)
			}
			if !bytes.Equal(dec, tc.Decoded) {
				t.Errorf("got %v, want %v", dec, tc.Decoded)
			}
		})
	}
}

func TestWriter(t *testing.T) {
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			buf := bytes.NewBuffer(make([]byte, 0, len(tc.Encoded)))
			d := NewDecoder(buf, WithSentinel(tc.Sentinel), WithReduced(tc.Reduced))
			e := NewEncoder(d, WithSentinel(tc.Sentinel), WithReduced(tc.Reduced))

			n, err := e.Write(tc.Decoded)
			if err != nil {
				t.Errorf("writer error: %v", err)
			}
			err = e.Close()
			if err != nil {
				t.Errorf("writer encoder close error: %v", err)
			}
			err = d.Close()
			if err != nil {
				t.Errorf("writer decoder close error: %v", err)
			}
			if d.NeedsMoreData() {
				t.Error("writer incomplete decode data")
			}
			if n != len(tc.Decoded) {
				t.Errorf("writer length got %d, want %d", n, len(tc.Decoded))
			}
			if !bytes.Equal(buf.Bytes(), tc.Decoded) {
				t.Errorf("got %v, want %v", buf.Bytes(), tc.Decoded)
			}
		})
	}
}

func TestStream(t *testing.T) {
	pr, pw := io.Pipe()
	testCases := testutil.Load(t, "testdata")

	go func() {
		defer pw.Close()

		for _, tc := range testCases {
			e := NewEncoder(pw,
				WithSentinel(tc.Sentinel),
				WithDelimiterOnClose(true),
				WithReduced(tc.Reduced))
			_, err := e.Write(tc.Decoded)
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
		d := NewDecoder(&buf, WithSentinel(tc.Sentinel), WithReduced(tc.Reduced))
		_, err := io.Copy(d, pr)

		if err != EOD {
			t.Error("stream decode EOD missing")
		}

		if d.NeedsMoreData() {
			t.Error("stream decode frame incomplete")
		}

		if !bytes.Equal(buf.Bytes(), tc.Decoded) {
			t.Errorf("stream decode got %v, want %v", buf.Bytes(), tc.Decoded)
		}

		buf.Reset()

		t.Logf("stream decode %s", displayName(tc))
	}
}

func FuzzEncodeDecode(f *testing.F) {
	testCases := testutil.Load(f, "testdata")
	for _, tc := range testCases {
		for s := 0; s <= 255; s++ {
			f.Add(tc.Decoded, byte(s), false)
			f.Add(tc.Decoded, byte(s), true)
		}
	}
	f.Fuzz(func(t *testing.T, a []byte, del byte, reduced bool) {
		enc, err := Encode(a, WithSentinel(del), WithReduced(reduced))
		if err != nil {
			t.Errorf("fuzz encode error: %v", err)
		}
		if i := bytes.IndexByte(enc, del); i != -1 {
			t.Errorf("fuzz encode %v has sentinel at %d", enc, i)
		}

		dec, err := Decode(enc, WithSentinel(del), WithReduced(reduced))
		if err != nil {
			t.Errorf("fuzz decode error: %v", err)
		}
		if !bytes.Equal(dec, a) {
			t.Errorf("fuzz decode got %v want %v", dec, a)
		}
	})
}

func FuzzChainWriter(f *testing.F) {
	testCases := testutil.Load(f, "testdata")
	for _, tc := range testCases {
		for del := 0; del <= 255; del++ {
			f.Add(tc.Decoded, byte(del), false)
			f.Add(tc.Decoded, byte(del), true)
		}
	}
	f.Fuzz(func(t *testing.T, a []byte, del byte, reduced bool) {
		var buf bytes.Buffer
		d := NewDecoder(&buf, WithSentinel(del), WithReduced(reduced))
		e := NewEncoder(d, WithSentinel(del), WithReduced(reduced))

		n, err := e.Write(a)
		if err != nil {
			t.Errorf("fuzz chain error: %v", err)
		}
		if n != len(a) {
			t.Errorf("fuzz chain length got %d want %d", n, len(a))
		}

		err = e.Close()
		if err != nil {
			t.Errorf("fuzz chain encoder close error: %v", err)
		}
		err = d.Close()
		if err != nil {
			t.Errorf("fuzz chain decoder close error: %v", err)
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
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			e := NewEncoder(
				&LimitedWriter{io.Discard, 1, io.EOF},
				WithSentinel(tc.Sentinel),
				WithReduced(tc.Reduced),
				WithDelimiterOnClose(true))

			_, err := e.Write(tc.Decoded)
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
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			// The empty string is expected to have no writes
			if len(tc.Decoded) == 0 {
				t.SkipNow()
			}

			d := NewDecoder(
				&LimitedWriter{io.Discard, int64(len(tc.Decoded) - 1), io.EOF},
				WithReduced(tc.Reduced),
				WithSentinel(tc.Sentinel))

			_, err := d.Write(tc.Encoded)
			// err can be nil if no groups have been flushed, append a delimiter
			if err == nil {
				err = d.WriteByte(tc.Sentinel)
			}
			if err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestDecodeErrorOnClose(t *testing.T) {
	testCases := testutil.Load(t, "testdata")
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", displayName(tc), tc.Sentinel), func(t *testing.T) {
			// The empty string is expected to have no writes
			if len(tc.Decoded) == 0 {
				t.SkipNow()
			}

			d := NewDecoder(
				&LimitedWriter{io.Discard, int64(len(tc.Decoded) - 1), io.EOF},
				WithReduced(tc.Reduced),
				WithSentinel(tc.Sentinel))

			_, err := d.Write(tc.Encoded)
			// err can be nil if no groups have been flushed, append a delimiter
			if err == nil {
				err = d.Close()
			}
			if err != io.EOF {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
