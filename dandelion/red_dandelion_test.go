package dandelion

import (
	"testing"
)

func TestRedDandelion_CreateAccount(t *testing.T) {
	dandelion, err := NewRedDandelion()
	if err != nil {
		t.Error(err)
	}
	err = dandelion.OpenDatabase()
	if err != nil {
		t.Error(err)
	}
	if err != nil {
		t.Error(err)
	}
	defer func() {
		err := dandelion.Clean()
		if err != nil {
			t.Error(err)
		}
	}()
}
