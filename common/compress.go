package common

import (
	"bytes"
	"compress/zlib"
)

func Compress(plain []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	if _, err := w.Write(plain); err != nil {
		return nil, err
	}
	_ = w.Close()
	return buf.Bytes(), nil
}

func Decompress(compressed []byte) ([]byte, error) {
	if r, err := zlib.NewReader(bytes.NewReader(compressed)); err != nil {
		return nil, err
	} else {
		buf := new(bytes.Buffer)
		if _, err = buf.ReadFrom(r); err != nil {
			return nil, err
		}
		_ = r.Close()
		return buf.Bytes(), nil
	}
}
