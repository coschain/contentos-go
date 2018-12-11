package common

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type ZeroCopySink struct {
	buf []byte
}

// tryGrowByReslice is a inlineable version of grow for the fast-case where the
// internal buffer only needs to be resliced.
// It returns the index where bytes should be written and whether it succeeded.
func (self *ZeroCopySink) tryGrowByReslice(n int) (int, bool) {
	if l := len(self.buf); n <= cap(self.buf)-l {
		self.buf = self.buf[:l+n]
		return l, true
	}
	return 0, false
}

const maxInt = int(^uint(0) >> 1)

// grow grows the buffer to guarantee space for n more bytes.
// It returns the index where bytes should be written.
// If the buffer can't grow it will panic with ErrTooLarge.
func (self *ZeroCopySink) grow(n int) int {
	// Try to grow by means of a reslice.
	if i, ok := self.tryGrowByReslice(n); ok {
		return i
	}

	l := len(self.buf)
	c := cap(self.buf)
	if c > maxInt-c-n {
		panic(ErrTooLarge)
	}
	// Not enough space anywhere, we need to allocate.
	buf := makeSlice(2*c + n)
	copy(buf, self.buf)
	self.buf = buf[:l+n]
	return l
}

func (self *ZeroCopySink) WriteBytes(p []byte) {
	data := self.NextBytes(uint64(len(p)))
	copy(data, p)
}

func (self *ZeroCopySink) Size() uint64 { return uint64(len(self.buf)) }

func (self *ZeroCopySink) NextBytes(n uint64) (data []byte) {
	m, ok := self.tryGrowByReslice(int(n))
	if !ok {
		m = self.grow(int(n))
	}
	data = self.buf[m:]
	return
}

// Backs up a number of bytes, so that the next call to NextXXX() returns data again
// that was already returned by the last call to NextXXX().
func (self *ZeroCopySink) BackUp(n uint64) {
	l := len(self.buf) - int(n)
	self.buf = self.buf[:l]
}

func (self *ZeroCopySink) WriteUint32(data uint32) {
	buf := self.NextBytes(4)
	binary.LittleEndian.PutUint32(buf, data)
}

// NewReader returns a new ZeroCopySink reading from b.
func NewZeroCopySink(b []byte) *ZeroCopySink {
	if b == nil {
		b = make([]byte, 0, 512)
	}
	return &ZeroCopySink{b}
}

func (self *ZeroCopySink) Bytes() []byte { return self.buf }

func (self *ZeroCopySink) Reset() { self.buf = self.buf[:0] }

var ErrTooLarge = errors.New("bytes.Buffer: too large")

// makeSlice allocates a slice of size n. If the allocation fails, it panics
// with ErrTooLarge.
func makeSlice(n int) []byte {
	// If the make fails, give a known error.
	defer func() {
		if recover() != nil {
			panic(bytes.ErrTooLarge)
		}
	}()
	return make([]byte, n)
}
