package block

import (
	"strings"
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestBlockLog(t *testing.T) {
	var blog BLog
	home, err := homedir.Dir()
	if err != nil {
		t.Error(err.Error())
	}
	blog.Remove(home)
	err = blog.Open(home)
	if err != nil {
		t.Error(err.Error())
	}

	var psb PhonySignedBlock
	err = blog.Append(&psb)
	if err != nil {
		t.Error(err.Error())
	}
	err = blog.Append(&psb)
	if err != nil {
		t.Error(err.Error())
	}
	err = blog.ReadBlock(&psb, 0)
	if err != nil {
		t.Error(err.Error())
	}
	if strings.Compare(psb.Data(), "hello0") != 0 {
		t.Error("Expect hello0 while got: ", psb.Data())
	}
	err = blog.ReadBlock(&psb, 1)
	if err != nil {
		t.Error(err.Error())
	}
	if strings.Compare(psb.Data(), "hello1") != 0 {
		t.Error("Expect hello1 while got: ", psb.Data())
	}
}
