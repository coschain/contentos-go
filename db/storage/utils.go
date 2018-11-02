package storage

import (
	"bytes"
	"encoding/gob"
)

// defines a database writing operation (put or delete)
type writeOp struct {
	Key, Value []byte
	Del        bool
}

func encodeWriteOp(op writeOp) []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	enc.Encode(op)
	return buf.Bytes()
}

func decodeWriteOp(data []byte) *writeOp {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	var op writeOp
	if err := dec.Decode(&op); err == nil {
		return &op
	}
	return nil
}
