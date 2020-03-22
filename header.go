package conio

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const hdrSz = 4

// flag bits
const (
	bClosed = (hdrSz*8 - 1) - iota // highest bit

	closedBit = uint32(1 << bClosed)

	flagsCount = 1 // count of flag bits
)

const (
	maxSzBits = (hdrSz*8 - flagsCount) // header Size - number of flag bits above
	maxSz     = 1<<maxSzBits - 1
	minSz     = 0
)

// binheader is the header of each transfer chunk.  We pack int and all flags
// into `hdrSz` bytes.
type binheader struct {
	size   int
	closed int
}

var endianness = binary.LittleEndian

var (
	errBufSize       = errors.New("invalid size")
	errInvalidDataSz = errors.New("reader returned less bytes than expected")
)

// newBinHeader creates a new binary header.
func newBinHeader(size int, closed bool) (hdr *binheader, err error) {
	defer func() {
		if r := recover(); r != nil {
			hdr = nil
			err = fmt.Errorf("%v", r)
		}
	}()
	hdr = &binheader{
		size: size,
	}
	if closed {
		hdr.closed = 1
	}
	hdr.sane()
	return
}

func (b *binheader) Bytes() []byte {
	b.sane()
	var buf [hdrSz]byte
	csz := b.size
	if b.IsClosed() {
		csz |= int(closedBit)
	}
	endianness.PutUint32(buf[:], uint32(csz))
	return buf[:]
}

func (b *binheader) sane() {
	if b.size < minSz || maxSz < b.size {
		panic("size overflow")
	}
}

// Size returns the size.
func (b *binheader) Size() int {
	return b.size
}

// IsClosed returns true if the transfer is in final state.
func (b *binheader) IsClosed() bool {
	return !(b.closed == 0)
}

// loadHeader loads the header from bytes.  It only uses hdrSz bytes, the rest
// is ignored.  It does not modify p.
func loadHeader(p []byte) (*binheader, error) {
	if len(p) < hdrSz {
		return nil, errBufSize
	}
	csz := endianness.Uint32(p[:hdrSz])
	return &binheader{
		closed: int(csz >> bClosed & 1),
		size:   int(csz & ^(closedBit)),
	}, nil
}

// readHeader reads the header from reader.  It works by reading hdrSz bytes,
// and converting them to header.
func readHeader(r io.Reader) (*binheader, error) {
	var buf [hdrSz]byte
	n, err := io.ReadFull(r, buf[:])
	if err != nil {
		return nil, err
	}
	if n != hdrSz {
		return nil, errInvalidDataSz
	}
	return loadHeader(buf[:])
}

// WriteTo writes serialised header to reader
func (b *binheader) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(b.Bytes())
	return int64(n), err
}

func must(hdr *binheader, err error) *binheader {
	if err != nil {
		panic(err)
	}
	return hdr
}
