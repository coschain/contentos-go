package db

import (
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"

	"contentos-go/db/blocklog"
)

func TestBlockLog(t *testing.T) {
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

	var msb MockSignedBlock
	msb.Set("hello0")
	err = blog.Append(&msb)
	if err != nil {
		t.Error(err.Error())
	}
	msb.Set("hello1")
	err = blog.Append(&msb)
	if err != nil {
		t.Error(err.Error())
	}
	err = blog.ReadBlock(&msb, 0)
	if err != nil {
		t.Error(err.Error())
	}
	if strings.Compare(msb.Data(), "hello0") != 0 {
		t.Error("Expect hello0 while got: ", msb.Data())
	}
	err = blog.ReadBlock(&msb, 1)
	if err != nil {
		t.Error(err.Error())
	}
	if strings.Compare(msb.Data(), "hello1") != 0 {
		t.Error("Expect hello1 while got: ", msb.Data())
	}
}
