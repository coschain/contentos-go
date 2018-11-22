package wallet

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEncryptDataAndDecryptData(t *testing.T) {
	myassert := assert.New(t)
	c, iv, err := EncryptData([]byte("hello world"), []byte("123456"))
	myassert.NoError(err)
	d, err := DecryptData(c, []byte("123456"), iv)
	myassert.Equal(d, []byte("hello world"))
}

func TestEncryptDataAndDecryptDataWithUncorrectPassphrase(t *testing.T) {
	myassert := assert.New(t)
	c, iv, err := EncryptData([]byte("hello world"), []byte("123456"))
	myassert.NoError(err)
	d, err := DecryptData(c, []byte("111111"), iv)
	myassert.NotEqual(d, []byte("hello world"))
}
