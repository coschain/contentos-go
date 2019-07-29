package contracts

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	. "github.com/coschain/contentos-go/dandelion"
	"testing"
)

type NativeTester struct {

}

func (tester *NativeTester) Test(t *testing.T, d *Dandelion) {
	t.Run("sha256", d.Test(tester.sha256))
}

func (tester *NativeTester) sha256(t *testing.T, d *Dandelion) {
	data := make([]byte, 16)
	for i := 0; i < 10; i++ {
		_, _ = rand.Reader.Read(data)
		sum := sha256.Sum256(data)
		ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.sha256 %v, %v", BytesToJson(data), BytesToJson(sum[:])))
	}
	sum := sha256.Sum256(nil)
	ApplyNoError(t, d, fmt.Sprintf("actor1: actor1.native_tester.sha256 [], %v", BytesToJson(sum[:])))
}
