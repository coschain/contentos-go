package common

import (
	"github.com/magiconair/properties/assert"
	"testing"
)

func TestChecksum(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5}
	checksum1 := Checksum(data)

	writer := NewChecksum()
	writer.Write(data)
	checksum2 := writer.Sum(nil)

	assert.Equal(t, checksum1[:], checksum2)
}