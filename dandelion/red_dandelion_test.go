package dandelion

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRedDandelion_CreateAccount(t *testing.T) {
	dandelion, err := NewRedDandelion()
	myassert := assert.New(t)
	if err != nil {
		t.Error(err)
	}
	err = dandelion.OpenDatabase()
	if err != nil {
		t.Error(err)
	}

	defer func() {
		err := dandelion.Clean()
		if err != nil {
			t.Error(err)
		}
	}()

	err = dandelion.CreateAccount("kochiya")
	if err != nil {
		t.Error(err)
	}

	acc := dandelion.GetAccount("kochiya")
	myassert.NotNil(acc)
	myassert.Equal(acc.GetName().Value, "kochiya")
}
