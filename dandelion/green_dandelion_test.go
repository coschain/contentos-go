package dandelion

import (
	"fmt"
	"github.com/inconshreveable/log15"
	"testing"
)

func TestGreenDandelion_CreateAccount(t *testing.T) {
	log := log15.New()
	dandelion, err := NewDandelion(log)
	if err != nil {
		log.Error("error:", err)
	}
	err = dandelion.OpenDatabase()
	if err != nil {
		log.Error("error:", err)
	}
	err = dandelion.CreateAccount("kochiya")
	if err != nil {
		log.Error("error:", err)
	}

	acc := dandelion.GetAccount("kochiya")
	if acc != nil {
		fmt.Println(acc.GetName())
	} else {
		fmt.Println("cannot find acc")
	}

	err = dandelion.Clean()
	if err != nil {
		log.Error("error:", err)
	}
}
