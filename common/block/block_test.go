package block

import (
	"testing"
)

func TestBlockLog(t *testing.T) {
	var blog BLog
	err := blog.Open("/Users/jesse")
	if err != nil {
		t.Error(err.Error())
	}
}
