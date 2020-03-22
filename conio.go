// Package conio provides controlled I/O reader and writer.  It may be useful
// when you need to have a compressed reader/writer over the net.Conn and then
// resume your normal reads and writes on it.
package conio

import (
	"fmt"
	"io"
)

// ConReader is the controlled reader.
type ConReader struct {
	r      io.Reader
	unread int
}

// ConReader is the controlled writer.  It must be closed after using.
type ConWriter struct {
	w io.Writer
}

// NewReader creates a new ConReader.
func NewReader(r io.Reader) *ConReader {
	return &ConReader{r: r}
}

// NewWriter creates a new ConWriter.
func NewWriter(w io.Writer) *ConWriter {
	return &ConWriter{w}
}

// Read reads the data from the underlying reader into p.
func (r *ConReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	if r.unread == 0 {
		hdr, err := readHeader(r.r)
		if err != nil {
			return 0, err
		}
		r.unread = hdr.Size()
		if hdr.IsClosed() {
			return 0, io.EOF
		}
	}
	if r.unread < len(p) {
		p = p[:r.unread]
	}
	n, err := r.r.Read(p)
	r.unread -= n
	return n, err
}

// Writes writes the data to the underlying reader.
func (w *ConWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	hdr, err := newBinHeader(len(p), false)
	if err != nil {
		return 0, err
	}
	if _, err := hdr.WriteTo(w.w); err != nil {
		return 0, err
	}
	return w.w.Write(p)
}

// Close closes the Writer.
func (w *ConWriter) Close() error {
	if _, err := must(newBinHeader(0, true)).WriteTo(w.w); err != nil {
		return fmt.Errorf("error closing writer: %w", err)
	}
	return nil
}
