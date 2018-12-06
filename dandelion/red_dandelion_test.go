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

func TestRedDandelion_Fund(t *testing.T) {
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

	_ = dandelion.CreateAccount("kochiya")
	err = dandelion.Fund("kochiya", 1000)
	if err != nil {
		t.Error(err)
	}
	acc := dandelion.GetAccount("kochiya")
	myassert.NotNil(acc)
	myassert.Equal(acc.GetBalance().Value, uint64(1000))
}
