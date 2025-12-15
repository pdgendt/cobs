// Package cobs implements Consistent Overhead Byte Stuffing (COBS) encoding and decoding algorithms
// for efficient, reliable and unambiguous packet framing.
package cobs

import (
	"bytes"
	"errors"
	"io"
)

const (
	Delimiter = byte(0x00) // default packet framing delimiter.
)

// EOD is the error returned when decoding and a delimiter was written.
// Functions return EOD to signal a graceful end of a frame.
var EOD = errors.New("EOD")

// ErrUnexpectedEOD means that a delimiter was encountered in a malformed frame.
var ErrUnexpectedEOD = errors.New("unexpected EOD")

// ErrIncompleteData means a decoder was closed with an incomplete frame.
var ErrIncompleteFrame = errors.New("frame incomplete")

// An Encoder implements the io.Writer and io.ByteWriter interfaces. Data
// written will we be encoded into groups and forwarded.
type Encoder struct {
	w         io.Writer
	buf       []byte
	delimiter byte
}

// A Decoder implements the io.Writer and io.ByteWriter interfaces. Data
// written will we be decoded and forwarded byte per byte.
type Decoder struct {
	w         io.Writer
	code      byte
	codeIndex byte
	delimiter byte
}

func maxCode(delimiter byte) byte {
	if delimiter == 0xff {
		return 0xfe
	}
	return 0xff
}

// NewEncoder returns an Encoder that writes encoded data to w using the default delimiter (0x00).
func NewEncoder(w io.Writer) *Encoder {
	return NewEncoderWithDelimiter(w, Delimiter)
}

// NewEncoderWithDelimiter returns an Encoder that writes encoded data to w using a custom delimiter.
func NewEncoderWithDelimiter(w io.Writer, delimiter byte) *Encoder {
	e := new(Encoder)

	e.w = w
	e.delimiter = delimiter
	// Create a buffer with maximum capacity for a group
	e.buf = make([]byte, 1, 255)
	e.buf[0] = 0
	e.checkLen()

	return e
}

func (e *Encoder) finish() error {
	if _, err := e.w.Write(e.buf); err != nil {
		return err
	}

	// reset buffer
	e.buf = e.buf[:1]
	e.buf[0] = 0
	e.checkLen()

	return nil
}

func (e *Encoder) checkLen() {
	if e.buf[0] == e.delimiter {
		e.buf[0]++
	}
}

// WriteByte encodes a single byte c. If a group is finished
// it is written to w.
func (e *Encoder) WriteByte(c byte) error {
	// Finish if group is full
	if e.buf[0] == maxCode(e.delimiter) {
		if err := e.finish(); err != nil {
			return err
		}
	}

	if c == e.delimiter {
		return e.finish()
	}

	e.buf = append(e.buf, c)
	e.buf[0]++
	e.checkLen()

	return nil
}

// Write will call WriteByte for each byte in p.
func (e *Encoder) Write(p []byte) (int, error) {
	for i, c := range p {
		if err := e.WriteByte(c); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

// Close has to be called after writing a full frame and
// will write the last group.
func (e *Encoder) Close() error {
	return e.finish()
}

// Encode encodes with the default delimiter (0x00) and returns a byte slice.
func Encode(data []byte) ([]byte, error) {
	return EncodeWithDelimiter(data, Delimiter)
}

// Encode encodes and returns a byte slice.
func EncodeWithDelimiter(data []byte, delimiter byte) ([]byte, error) {
	// Reserve a buffer with overhead room
	buf := bytes.NewBuffer(make([]byte, 0, len(data)+(len(data)+253)/254))
	e := NewEncoderWithDelimiter(buf, delimiter)

	if _, err := e.Write(data); err != nil {
		return buf.Bytes(), err
	}

	err := e.Close()

	return buf.Bytes(), err
}

// NewDecoder returns a Decoder that writes decoded data to w using the default delimiter (0x00).
func NewDecoder(w io.Writer) *Decoder {
	return NewDecoderWithDelimiter(w, Delimiter)
}

// NewDecoderWithDelimiter returns a Decoder that writes decoded data to w using a custom delimiter.
func NewDecoderWithDelimiter(w io.Writer, delimiter byte) *Decoder {
	d := new(Decoder)

	d.w = w
	d.delimiter = delimiter
	d.code = maxCode(d.delimiter)
	d.codeIndex = 0

	return d
}

// WriteByte decodes a single byte c. If c is a delimiter the decoder
// state is validated and either EOD or ErrUnexpectedEOD is returned.
func (d *Decoder) WriteByte(c byte) error {
	// Got a delimiter
	if c == d.delimiter {
		if d.codeIndex != 0 {
			return ErrUnexpectedEOD
		}

		// Reset state
		d.code = maxCode(d.delimiter)

		return EOD
	}

	if d.codeIndex > 0 {
		if _, err := d.w.Write([]byte{c}); err != nil {
			return err
		}
		d.codeIndex--

		return nil
	}

	d.codeIndex = c

	if d.code != maxCode(d.delimiter) {
		if _, err := d.w.Write([]byte{d.delimiter}); err != nil {
			return err
		}
	}

	d.code = d.codeIndex
	// If our encoded length is larger than the delimiter, we need to subtract one.
	if d.codeIndex > d.delimiter {
		d.codeIndex--
	}

	return nil
}

// Write will call WriteByte for each byte in p.
func (d *Decoder) Write(p []byte) (int, error) {
	for i, c := range p {
		if err := d.WriteByte(c); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

// NeedsMoreData returns true if the decoder needs more data for a full frame.
func (d *Decoder) NeedsMoreData() bool {
	return d.codeIndex != 0
}

// Decode decodes with the default delimiter (0x00) and returns a byte slice.
func Decode(data []byte) ([]byte, error) {
	return DecodeWithDelimiter(data, Delimiter)
}

// Decode decodes and returns a byte slice.
func DecodeWithDelimiter(data []byte, delimiter byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)))
	d := NewDecoderWithDelimiter(buf, delimiter)

	if _, err := d.Write(data); err != nil {
		return buf.Bytes(), err
	}

	if d.NeedsMoreData() {
		return buf.Bytes(), ErrIncompleteFrame
	}

	return buf.Bytes(), nil
}
