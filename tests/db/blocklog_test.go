package db

import (
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/coschain/contentos-go/db/blocklog"
	"github.com/mitchellh/go-homedir"
)

func TestBlockLog(t *testing.T) {
	assert := assert.New(t)
	var blog blocklog.BLog
	home, err := homedir.Dir()
	if err != nil {
		t.Error(err.Error())
	}
	blog.Remove(home)
	err = blog.Open(home)
	if err != nil {
		t.Error(err.Error())
	}

	assert.Equal(blog.Empty(), true)
	var msb MockSignedBlock
	msb.Payload = []byte("hello0")
	assert.NoError(blog.Append(&msb))

	assert.Equal(blog.Empty(), false)

	msb.Payload = []byte("hello1")
	assert.NoError(blog.Append(&msb))

	assert.NoError(blog.ReadBlock(&msb, 0))
	assert.Equal(msb.Data(), "hello0")

	assert.NoError(blog.ReadBlock(&msb, 1))
	assert.Equal(msb.Data(), "hello1")

}
