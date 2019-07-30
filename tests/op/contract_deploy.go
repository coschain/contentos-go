package op

import (
	"crypto/rand"
	"fmt"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

type ContractDeployTester struct {}

func (tester *ContractDeployTester) Test(t *testing.T, d *Dandelion) {
	if err := StakeFund(d, 3); err != nil {
		t.Fatal(err)
	}
	t.Run("deployForOthers", d.Test(tester.deployForOthers))
	t.Run("invalidFormats", d.Test(tester.invalidFormats))
	t.Run("hasFloats", d.Test(tester.hasFloats))
	t.Run("unknownImports", d.Test(tester.unknownImports))
	t.Run("upgradable", d.Test(tester.upgradable))
}

func (tester *ContractDeployTester) deployForOthers(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.Error(tester.deploy(d, "actor0", "actor1", "native_tester", true))
	a.Error(tester.deploy(d, "actor0", "xxxxxxx", "native_tester", true))
}

func (tester *ContractDeployTester) invalidFormats(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	code := make([]byte, 100000)
	abi := make([]byte, 100000)
	for i := 0; i < 10; i++ {
		_, _ = rand.Reader.Read(code)
		_, _ = rand.Reader.Read(abi)
		name := fmt.Sprintf("contract%d", i)
		a.Error(tester.deployUncompressed(d, "actor0", "actor0", name, nil, nil, true))
		a.Error(tester.deployCompressed(d, "actor0", "actor0", name + "_c", nil, nil, true))
	}
}

func (tester *ContractDeployTester) hasFloats(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.Error(tester.deploy(d, "actor0", "actor0", "has_float", true))
}

func (tester *ContractDeployTester) unknownImports(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	a.Error(tester.deploy(d, "actor0", "actor0", "unknown_imports", true))
}

func (tester *ContractDeployTester) upgradable(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	var (
		code1, abi1, code2, abi2 []byte
		err error
	)
	// 2 versions of hello contract
	code1, abi1, err = ContractCodeAndAbi("hello")
	a.NoError(err)
	code2, abi2, err = ContractCodeAndAbi("hello2")
	a.NoError(err)

	// actor1.hello is non-upgradable
	a.NoError(tester.deployUncompressed(d, "actor1", "actor1", "hello", code1, abi1, false))
	// updating non-upgradable contracts always fails
	a.Error(tester.deployUncompressed(d, "actor1", "actor1", "hello", code1, abi1, true))
	a.Error(tester.deployUncompressed(d, "actor1", "actor1", "hello", code2, abi2, true))
	a.Error(tester.deployUncompressed(d, "actor1", "actor1", "hello", code1, abi1, false))
	a.Error(tester.deployUncompressed(d, "actor1", "actor1", "hello", code2, abi2, false))

	// actor0.hello is upgradable
	a.NoError(tester.deployUncompressed(d, "actor0", "actor0", "hello", code1, abi1, true))
	// you can't update a contract without any changes.
	a.Error(tester.deployUncompressed(d, "actor0", "actor0", "hello", code1, abi1, true))
	// update to version 2
	a.NoError(tester.deployUncompressed(d, "actor0", "actor0", "hello", code2, abi2, true))
	// update back to version 1, and make itself non-upgradable.
	a.NoError(tester.deployUncompressed(d, "actor0", "actor0", "hello", code1, abi1, false))
	// later updates never succeed
	a.Error(tester.deployUncompressed(d, "actor0", "actor0", "hello", code2, abi2, true))
	a.Error(tester.deployUncompressed(d, "actor0", "actor0", "hello", code2, abi2, false))
	a.Error(tester.deployUncompressed(d, "actor0", "actor0", "hello", code1, abi1, true))
	a.Error(tester.deployUncompressed(d, "actor0", "actor0", "hello", code1, abi1, false))
}


//
// helper methods.
//

func (tester *ContractDeployTester) deploy(d *Dandelion, signer, owner, contract string, upgradable bool) (err error) {
	var (
		code, abi []byte
	)
	if code, abi, err = ContractCodeAndAbi(contract); err != nil {
		return err
	}
	return tester.deployUncompressed(d, signer, owner, contract, code, abi, upgradable)
}

func (tester *ContractDeployTester) deployUncompressed(d *Dandelion, signer, owner, contract string, code, abi []byte, upgradable bool) (err error) {
	r := d.Account(signer).TrxReceipt(ContractDeployUncompressed(owner, contract, abi, code, upgradable, "", ""))
	if r == nil || r.Status != prototype.StatusSuccess {
		err = fmt.Errorf("contract deployment failed, receipt = %v", r)
	}
	return
}

func (tester *ContractDeployTester) deployCompressed(d *Dandelion, signer, owner, contract string, code, abi []byte, upgradable bool) (err error) {
	r := d.Account(signer).TrxReceipt(ContractDeploy(owner, contract, abi, code, upgradable, "", ""))
	if r == nil || r.Status != prototype.StatusSuccess {
		err = fmt.Errorf("contract deployment failed, receipt = %v", r)
	}
	return
}
