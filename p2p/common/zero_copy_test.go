package common

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestSourceSink(t *testing.T) {
	sink1 := NewZeroCopySink(nil)
	a1 := uint32(4294967295)
	sink1.WriteUint32(a1)
	source1 := NewZeroCopySource(sink1.Bytes())
	assert.Equal(t, len(sink1.Bytes()), int(source1.Size()) )
	assert.Equal(t, sink1.Bytes(), source1.Data() )

	sink2 := NewZeroCopySink(nil)
	a2 := []byte{8, 18, 88}
	sink2.WriteBytes(a2)
	source2 := NewZeroCopySource(sink2.Bytes())
	assert.Equal(t, len(sink2.Bytes()), int(source2.Size()) )
	assert.Equal(t, sink2.Bytes(), source2.Data())
}