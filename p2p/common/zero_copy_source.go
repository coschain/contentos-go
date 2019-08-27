package common

type ZeroCopySource struct {
	s   []byte
	off uint64 // current reading index
}

// Len returns the number of bytes of the unread portion of the
// slice.
//func (self *ZeroCopySource) Len() uint64 {
//	length := uint64(len(self.s))
//	if self.off >= length {
//		return 0
//	}
//	return length - self.off
//}

func (self *ZeroCopySource) Data() []byte {
	return self.s
}

//func (self *ZeroCopySource) Pos() uint64 {
//	return self.off
//}

// Size returns the original length of the underlying byte slice.
// Size is the number of bytes available for reading via ReadAt.
// The returned value is always the same and is not affected by calls
// to any other method.
//func (self *ZeroCopySource) Size() uint64 { return uint64(len(self.s)) }

// NewReader returns a new ZeroCopySource reading from b.
func NewZeroCopySource(b []byte) *ZeroCopySource { return &ZeroCopySource{b, 0} }
