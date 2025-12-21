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

type config struct {
	sentinel         byte
	delimiterOnClose bool
	reduced          bool
}

type option func(*config)

// WithSentinel configures the encoder/decoder to use a custom sentinel value.
func WithSentinel(sentinel byte) option {
	return func(c *config) {
		c.sentinel = sentinel
	}
}

// WithReduced configures the encoder/decoder to run COBS/R.
func WithReduced(enabled bool) option {
	return func(c *config) {
		c.reduced = enabled
	}
}

// WithDelimiterOnClose configures the encoder to append a sentinel delimiter on close.
func WithDelimiterOnClose(enabled bool) option {
	return func(c *config) {
		c.delimiterOnClose = enabled
	}
}

// An Encoder implements the io.Writer and io.ByteWriter interfaces. Data
// written will we be encoded into groups and forwarded.
type Encoder struct {
	config
	w   io.Writer
	buf []byte
}

// A Decoder implements the io.Writer and io.ByteWriter interfaces. Data
// written will we be decoded and forwarded byte per byte.
type Decoder struct {
	config
	w         io.Writer
	code      byte
	codeIndex byte
}

// NewEncoder returns an Encoder that writes encoded data to w.
func NewEncoder(w io.Writer, opts ...option) *Encoder {
	e := &Encoder{
		config: config{sentinel: Delimiter},
		w:      w,
		// Create a buffer with maximum capacity for a group
		buf: make([]byte, 1, 255),
	}
	for _, opt := range opts {
		opt(&e.config)
	}

	e.buf[0] = 1

	return e
}

func (e *Encoder) finish(close bool) error {
	// From https://pythonhosted.org/cobs/cobsr-intro.html
	// Replace the overhead byte with the last data byte if it is larger than
	// the running buffer size.
	if close && e.reduced && len(e.buf) > 1 && e.buf[0] < e.buf[len(e.buf)-1] {
		// Put the last element as the overhead byte
		e.buf[0] = e.buf[len(e.buf)-1]
		e.buf = e.buf[:len(e.buf)-1]
	}

	if e.sentinel != Delimiter {
		for i := range e.buf {
			e.buf[i] ^= e.sentinel
		}
	}

	if _, err := e.w.Write(e.buf); err != nil {
		return err
	}

	if close && e.delimiterOnClose {
		if _, err := e.w.Write([]byte{e.sentinel}); err != nil {
			return err
		}
	}

	// reset buffer
	e.buf = e.buf[:1]
	e.buf[0] = 1

	return nil
}

// WriteByte encodes a single byte c. If a group is finished
// it is written to w.
func (e *Encoder) WriteByte(c byte) error {
	// Finish if group is full
	if e.buf[0] == 0xff {
		if err := e.finish(false); err != nil {
			return err
		}
	}

	if c == Delimiter {
		return e.finish(false)
	}

	e.buf = append(e.buf, c)
	e.buf[0]++

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
	return e.finish(true)
}

// Encode encodes and returns a byte slice.
func Encode(data []byte, opts ...option) ([]byte, error) {
	// Reserve a buffer with overhead room
	buf := bytes.NewBuffer(make([]byte, 0, len(data)+(len(data)+253)/254))
	e := NewEncoder(buf, opts...)

	if _, err := e.Write(data); err != nil {
		return buf.Bytes(), err
	}

	err := e.Close()

	return buf.Bytes(), err
}

// NewDecoder returns a Decoder that writes decoded data to w.
func NewDecoder(w io.Writer, opts ...option) *Decoder {
	d := &Decoder{
		config:    config{sentinel: Delimiter},
		w:         w,
		codeIndex: 0,
		code:      0xff,
	}
	for _, opt := range opts {
		opt(&d.config)
	}

	return d
}

func (d *Decoder) flushReduced() error {
	// Check if we are decoding a reduced buffer
	if d.reduced && d.codeIndex > 0 {
		_, err := d.w.Write([]byte{d.code})
		if err != nil {
			return err
		}
		d.codeIndex = 0
	}

	return nil
}

// WriteByte decodes a single byte c. If c is the sentinel value the decoder
// state is validated and either EOD or ErrUnexpectedEOD is returned.
func (d *Decoder) WriteByte(c byte) error {
	// XOR with the sentinel first
	c ^= d.sentinel

	// Got a sentinel
	if c == Delimiter {
		err := d.flushReduced()
		if err != nil {
			return err
		}

		if d.codeIndex != 0 {
			return ErrUnexpectedEOD
		}

		// Reset state
		d.code = 0xff

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

	if d.code != 0xff {
		if _, err := d.w.Write([]byte{Delimiter}); err != nil {
			return err
		}
	}

	d.code = d.codeIndex
	d.codeIndex--

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

// Close flushes the last byte in case of COBS reduced (COBS/R) and will
// return an error if the decoder expects more data.
func (d *Decoder) Close() error {
	err := d.flushReduced()
	if err != nil {
		return err
	}

	if d.NeedsMoreData() {
		return ErrIncompleteFrame
	}

	return nil
}

// Decode decodes and returns a byte slice.
func Decode(data []byte, opts ...option) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)))
	d := NewDecoder(buf, opts...)

	if _, err := d.Write(data); err != nil {
		return buf.Bytes(), err
	}

	err := d.Close()

	return buf.Bytes(), err
}
