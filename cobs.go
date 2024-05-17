package cobs

import (
	"bytes"
	"errors"
	"io"
)

const (
	Delimiter = byte(0x00)
)

var EOD = errors.New("EOD")

var ErrUnexpectedEOD = errors.New("unexpected EOD")

type Encoder struct {
	w   io.Writer
	buf []byte
}

type Decoder struct {
	w         io.Writer
	code      byte
	codeIndex byte
}

func NewEncoder(w io.Writer) *Encoder {
	e := new(Encoder)

	e.w = w
	// Create a buffer with maximum capacity for a block
	e.buf = make([]byte, 1, 255)
	e.buf[0] = 1

	return e
}

func (e *Encoder) finish() (err error) {
	for i, n := 0, 0; i < len(e.buf); i += n {
		if n, err = e.w.Write(e.buf[i:]); err != nil {
			return err
		}
	}

	// reset buffer
	e.buf = e.buf[:1]
	e.buf[0] = 1

	return nil
}

func (e *Encoder) WriteByte(c byte) error {
	// Finish if block is full
	if e.buf[0] == 0xff {
		if err := e.finish(); err != nil {
			return err
		}
	}

	if c == Delimiter {
		return e.finish()
	}

	e.buf = append(e.buf, c)
	e.buf[0]++

	return nil
}

func (e *Encoder) Write(p []byte) (int, error) {
	for i, c := range p {
		if err := e.WriteByte(c); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

func (e *Encoder) Close() error {
	return e.finish()
}

func Encode(data []byte) (enc []byte, err error) {
	// Reserve a buffer with overhead room
	buf := bytes.NewBuffer(make([]byte, 0, len(data) + (len(data) + 253) / 254))
	e := NewEncoder(buf)

	if _, err = e.Write(data); err != nil {
		return buf.Bytes(), err
	}

	err = e.Close()

	return buf.Bytes(), err
}

func NewDecoder(w io.Writer) *Decoder {
	d := new(Decoder)

	d.w = w
	d.code = 0xff
	d.codeIndex = 0

	return d
}

func (d *Decoder) WriteByte(c byte) error {
	// Got a delimiter
	if c == Delimiter {
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

func (d *Decoder) Write(p []byte) (int, error) {
	for i, c := range p {
		if err := d.WriteByte(c); err != nil {
			return i, err
		}
	}

	return len(p), nil
}

func Decode(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, len(data)))
	d := NewDecoder(buf)

	_, err := d.Write(data)

	return buf.Bytes(), err
}
