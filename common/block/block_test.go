package block

import (
	"testing"

	"github.com/mitchellh/go-homedir"
)

func TestBlockLog(t *testing.T) {
	var blog BLog
	home, err := homedir.Dir()
	if err != nil {
		t.Error(err.Error())
	}
	err = blog.Open(home)
	if err != nil {
		t.Error(err.Error())
	}
}
