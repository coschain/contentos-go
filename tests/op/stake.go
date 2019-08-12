package op

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/stretchr/testify/assert"
	"math"
	"math/big"
	"testing"
)

type StaminaType int

const (
	FREE StaminaType = 0
	STAKE StaminaType = 1
)

type StakeTester struct {
	acc0, acc1, acc2 *DandelionAccount
	rc utils.IResourceLimiter
}

func (tester *StakeTester) Test(t *testing.T, d *Dandelion) {
	tester.acc0 = d.Account("actor0")
	tester.acc1 = d.Account("actor1")
	tester.acc2 = d.Account("actor2")
	tester.rc   = utils.NewResourceLimiter()

	t.Run("normal", d.Test(tester.normal))
	t.Run("stake wrong sender", d.Test(tester.wrongSender))
	t.Run("stake wrong receiver", d.Test(tester.wrongReceiver))
	t.Run("stake wrong sender and receiver", d.Test(tester.wrongSenderAndReceiver))
	t.Run("stake zero", d.Test(tester.amountZero))
	t.Run("stake insufficient Balance", d.Test(tester.insufficientBalance))
	t.Run("multiple Stake", d.Test(tester.multipleStake))
	t.Run("modify global staminaFree", d.Test(tester.mdGlobalStaminaFree))
	t.Run("only consume free stamina", d.Test(tester.onlyConsumeFreeStamina))
	t.Run("consume stake stamina", d.Test(tester.consumeStakeStamina))
	t.Run("unable send trx", d.Test(tester.unableSendTrx))
	t.Run("regain to max stamina", d.Test(tester.regainToMaxStamina))
	t.Run("regain when produce new block", d.Test(tester.regain))
	t.Run("regain with same speed", d.Test(tester.sameRegainSpeed))
	t.Run("regain with different speed", d.Test(tester.diffRegainSpeed))
	t.Run("apply trx fail", d.Test(tester.applyTrxFail))
}


func (tester *StakeTester) normal(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	balance0 := tester.acc0.GetBalance().Value
	stakeVest1 := tester.acc1.GetStakeVest().Value
	a.NoError(tester.acc0.SendTrx(Stake(tester.acc0.Name, tester.acc1.Name, 100)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance0-100, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest1+100, tester.acc1.GetStakeVest().Value)

	//stake user's stake stamina
	// (stakeVest/GlobalDynamicData.stakeVest)*StakeVestGlobalDynamicData.OneDayStamina)
	maxStakeStamina := d.CalculateUserMaxStamina(tester.acc1.Name)
	dgp := d.GlobalProps()
	_,stakeStamina1 := tester.rc.GetStakeLeft(tester.acc1.GetStamina(), tester.acc1.GetStaminaFreeUseBlock(), dgp.HeadBlockNumber, maxStakeStamina)
	//has not use any stamina, so use's left stake stamina should be equal to user's max stake stamina
    a.Equal(maxStakeStamina, stakeStamina1)
}

func (tester *StakeTester) wrongSender(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	sender := d.Account("account1")
	a.Empty(sender.CheckExist())
	stakeVest2 := tester.acc2.GetStakeVest().Value
	a.Error(tester.acc0.SendTrx(Stake(sender.Name, tester.acc2.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(stakeVest2, tester.acc2.GetStakeVest().Value)
}


func (tester *StakeTester) wrongReceiver(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	receiver := d.Account("account2")
	a.Empty(receiver.CheckExist())
	balance1 := tester.acc1.GetBalance().Value
	a.Error(tester.acc0.SendTrx(Stake(tester.acc1.Name, receiver.Name, 1)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance1, tester.acc1.GetBalance().Value)
}

func (tester *StakeTester) wrongSenderAndReceiver(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	sender := d.Account("account3")
	a.Empty(sender.CheckExist())
	receiver := d.Account("account4")
	a.Empty(receiver.CheckExist())
	a.Error(tester.acc2.SendTrx(Stake(sender.Name, receiver.Name, 10)))
	a.NoError(d.ProduceBlocks(1))
	a.Nil(sender.GetBalance())
	a.Nil(receiver.GetStakeVest())

}

func (tester *StakeTester) amountZero(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance0 := tester.acc0.GetBalance().Value
	stakeVest2 := tester.acc2.GetStakeVest().Value
	a.Error(tester.acc1.SendTrx(Stake(tester.acc0.Name, tester.acc2.Name, 0)))
	a.Equal(balance0, tester.acc0.GetBalance().Value)
	a.Equal(stakeVest2, tester.acc2.GetStakeVest().Value)

}

func (tester *StakeTester) insufficientBalance(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	balance2 := tester.acc2.GetBalance().Value
	stakeVest0 := tester.acc0.GetStakeVest().Value
	a.NoError(tester.acc2.SendTrx(Stake(tester.acc2.Name, tester.acc0.Name, math.MaxUint64)))
	a.NoError(d.ProduceBlocks(1))
	a.Equal(balance2, tester.acc2.GetBalance().Value)
	a.Equal(stakeVest0, tester.acc0.GetStakeVest().Value)

}

func (tester *StakeTester) multipleStake(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctList := []*DandelionAccount{tester.acc0, tester.acc1, tester.acc2}
	listLen := len(acctList)

	for i := 0; i < 6; i++ {
		balance0 := tester.acc0.GetBalance().Value
		acct := acctList[i%listLen]
		stakeVest := acct.GetStakeVest().Value
		amount := uint64(20*(i+1))
		a.NoError(tester.acc0.SendTrx(Stake(tester.acc0.Name, acct.Name, amount)))
		a.NoError(d.ProduceBlocks(1))
		a.Equal(balance0-amount, tester.acc0.GetBalance().Value)
		a.Equal(stakeVest+amount, acct.GetStakeVest().Value)
	}

}

func (tester *StakeTester) mdGlobalStaminaFree(t *testing.T, d *Dandelion) {
	//user's max free stamina should be equal to GlobalProps StaminaFree
	a := assert.New(t)
	acctName := "account1"
	tester.createNewAccount(t, d, acctName, 10000)
	newAcct := d.Account(acctName)
	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	_,acctFreeStamina := tester.rc.GetFreeLeft(sysFreeStamina, newAcct.GetStamina(), newAcct.GetStaminaFreeUseBlock(), dgp.HeadBlockNumber)
	//has not use any stamina, so use's left free stamina should be equal to user's max free stamina
	a.Equal(sysFreeStamina, acctFreeStamina)
	//modify GlobalProps StaminaFree
	newSysFreeStamina := sysFreeStamina + 100
	tester.mdFreeStamina(t, d, newSysFreeStamina)
	_,newAcctFreeStamina := tester.rc.GetFreeLeft(newSysFreeStamina, newAcct.GetStamina(), newAcct.GetStaminaFreeUseBlock(), dgp.HeadBlockNumber)
	//after modify GlobalProps StaminaFree
	a.Equal(newAcctFreeStamina, newSysFreeStamina)
	//modify GlobalProps StaminaFree to origin value
	tester.mdFreeStamina(t, d, sysFreeStamina)
}


func (tester *StakeTester) onlyConsumeFreeStamina(t *testing.T, d *Dandelion) {
	//if user's free stamina is enough to deduct trx stamina,stake stamina will not be consumed
	a := assert.New(t)
	acctName := "actor3"
	tester.createNewAccount(t, d, acctName, 0)
	maxFreeStamina,letFreeStamina,_,_ := getUserStamina(acctName, d)
    a.Equal(maxFreeStamina, letFreeStamina)
	consumeStamina := tester.multipleTransfer(t, d, acctName, 1)
	maxFreeStamina,letFreeStamina,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)
	a.Condition(func() (success bool) {
		return consumeStamina <= maxFreeStamina
	})
	a.Equal(d.Account(acctName).GetStamina(), uint64(0))
	a.Equal(maxStakeStamina, leftStakeStamina)
	a.NotEqual(maxFreeStamina, letFreeStamina)

}

func (tester *StakeTester) consumeStakeStamina(t *testing.T, d *Dandelion) {
	//if user's free stamina is not enough to deduct trx stamina,stake stamina will be consumed
	a := assert.New(t)
	acctName := "actor4"
	tester.createNewAccount(t, d, acctName, 10000)

	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)

	consumeStamina := tester.multipleTransfer(t, d, acctName, 2)
	_,_,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)
	newAcct := d.Account(acctName)
    consumeStake := newAcct.GetStamina()

	consumeFree := newAcct.GetStaminaFree()
	a.Condition(func() (success bool) {
		return consumeStamina >  newFreeStamina
	})

	a.Equal(newFreeStamina, consumeFree)
	a.NotEqual(maxStakeStamina, leftStakeStamina)
	a.Condition(func() (success bool) {
		return consumeStake > uint64(0)
	})
	tester.mdFreeStamina(t, d, sysFreeStamina)
}

//
//once stamina be consumed, after produce constants.WindowSize blocks, the stamina should regain to max value
//
func (tester *StakeTester) regainToMaxStamina(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctName := "account2"
    tester.createNewAccount(t, d, acctName, 10000)

	//modify system staminaFree
	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)

	//get current free stamina of newAcct
	_,acctFreeStamina,maxStakeStamina,acctStakeStamina := getUserStamina(acctName, d)
	a.Equal(acctFreeStamina, newFreeStamina)
	a.Equal(maxStakeStamina, acctStakeStamina)

	//create trx to consume stamina
	consumeStamina := tester.multipleTransfer(t, d, acctName, 2)
	//get new free stamina and stake stamina after create trx
	_,_,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)
	//if free stamina is not enough to deduct consume, stake stamina will be deducted
	if consumeStamina > newFreeStamina {
		a.NotEqual(maxStakeStamina, leftStakeStamina)
	} else {
		a.Equal(maxStakeStamina, leftStakeStamina)
	}
	//after produce constants.WindowSize blocks, user's free stamina and stake stamina should regain to max value
	d.ProduceBlocks(constants.WindowSize)
	sysFreeStamina,leftFreeStamina,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)

    a.Equal(maxStakeStamina, leftStakeStamina)
	a.Equal(leftFreeStamina, sysFreeStamina)
	//modify system stamina to the original value
	tester.mdFreeStamina(t, d, sysFreeStamina)
}


func (tester *StakeTester) regain(t *testing.T, d *Dandelion) {
	//when the stamina is consumed, should be regained as new blocks are generated,the rate should be
	// 1-(newBlockNum-lastConsumeBlockNum)/constants.WindowSize
	a := assert.New(t)
	acctName := "account3"
	tester.createNewAccount(t, d, acctName, 100)

	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)
	consumeStamina := tester.multipleTransfer(t, d, acctName,2)
	newAcct := d.Account(acctName)
	consumeFree := newAcct.GetStaminaFree()
	consumeStake := newAcct.GetStamina()

	_,_,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)
	if consumeStamina <= newFreeStamina {
		a.Equal(maxStakeStamina, leftStakeStamina)
	}

	newBlkCnt := 100
	if constants.WindowSize < 1000 {
		newBlkCnt = constants.WindowSize/2
	}
	a.Condition(func() (success bool) {
		return newBlkCnt < constants.WindowSize
	})


	//calculate new consume after regain
	newConsumeFree := tester.calNewConsume(consumeFree, uint64(newBlkCnt) , constants.WindowSize)
	newConsumeStake := tester.calNewConsume(consumeStake,uint64(newBlkCnt), constants.WindowSize)

	a.NoError(d.ProduceBlocks(newBlkCnt))

	regainFree := getStaminaRegain(acctName, FREE, d)
	regainStake := getStaminaRegain(acctName, STAKE, d)
	//total consume = regain + current consume
	a.Equal(consumeFree-regainFree, newConsumeFree)
	a.Equal(consumeStake-regainStake, newConsumeStake)
	tester.mdFreeStamina(t, d, sysFreeStamina)
}

//
// if several accounts stake the same cos, after consuming the same endurance value, should be
// regained at the same rate
//
func (tester *StakeTester) sameRegainSpeed(t *testing.T, d *Dandelion) {
	a := assert.New(t)

	acctName4 := "account4"
	tester.createNewAccount(t, d, acctName4, 1000)
	newAcct4 := d.Account(acctName4)

	acctName5 := "account5"
	tester.createNewAccount(t, d, acctName5, 1000)
	newAcct5 := d.Account(acctName5)
	dgp := d.GlobalProps()

	a.Equal(newAcct4.GetStakeVest().Value, newAcct5.GetStakeVest().Value)
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)

	var transferAmount uint64 = 100
	a.NoError(tester.sendTransferTrx(newAcct4, 2, transferAmount))
	a.NoError(tester.sendTransferTrx(newAcct5, 2 , transferAmount))
	a.NoError(d.ProduceBlocks(1))

	dgp = d.GlobalProps()
	_,leftFreeStamina4,_,leftStakeStamina4 := getUserStamina(acctName4, d)
	_,leftFreeStamina5,_,leftStakeStamina5 := getUserStamina(acctName5, d)
	a.Equal(leftFreeStamina4, leftFreeStamina5)
	a.Equal(leftStakeStamina4, leftStakeStamina5)
	//after producing the same blocks, newAcct4 and newAcct5 be regained at the same rate
	newBlk := constants.WindowSize/80
	a.Condition(func() (success bool) {
		return newBlk < constants.WindowSize
	})
	a.NoError(d.ProduceBlocks(newBlk))

	_,curFreeStamina4,_,curStakeStamina4 := getUserStamina(acctName4, d)
	_,curFreeStamina5,_,curStakeStamina5 := getUserStamina(acctName5, d)

	a.NotEqual(curFreeStamina4, leftFreeStamina4)
	a.NotEqual(curFreeStamina5, leftFreeStamina5)
	a.Equal(curFreeStamina4, curFreeStamina5)
	a.NotEqual(curStakeStamina4, leftStakeStamina4)
	a.NotEqual(curStakeStamina5, leftStakeStamina5)
	a.Equal(curStakeStamina4, curStakeStamina5)
	tester.mdFreeStamina(t, d, sysFreeStamina)
}


//
// if several accounts stake the same cos, after consuming the same endurance value, should be
// regained at the different rate
//
func (tester *StakeTester) diffRegainSpeed(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctName6 := "account6"
	tester.createNewAccount(t, d, acctName6, 1000)
	newAcct6 := d.Account(acctName6)

	acctName7 := "account7"
	tester.createNewAccount(t, d, acctName7, 1000)
	newAcct7 := d.Account(acctName7)
	a.Equal(newAcct6.GetStakeVest().Value, newAcct7.GetStakeVest().Value)
	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)

	var transferAmount uint64 = 100
	a.NoError(tester.sendTransferTrx(newAcct6, 1, transferAmount))
	a.NoError(tester.sendTransferTrx(newAcct7, 3 , transferAmount))
	a.NoError(d.ProduceBlocks(1))

	_,leftFreeStamina6,_,leftStakeStamina6 := getUserStamina(acctName6, d)
	_,leftFreeStamina7,_,leftStakeStamina7 := getUserStamina(acctName7, d)
	consumeStake6 := newAcct6.GetStamina()
	consumeStake7 := newAcct7.GetStamina()
    a.Condition(func() (success bool) {
		return consumeStake6 < consumeStake7
	})

	a.Condition(func() (success bool) {
		return leftStakeStamina6 > leftStakeStamina7
	})

	newBlk := constants.WindowSize/80
	a.Condition(func() (success bool) {
		return newBlk < constants.WindowSize
	})
	//after producing the same blocks, newAcct6 and newAcct6 be regained at the different rate,
	//account6 regain slower than account7
	a.NoError(d.ProduceBlocks(newBlk))
	regainStake6 := getStaminaRegain(acctName6, STAKE, d)
	regainStake7 := getStaminaRegain(acctName7, STAKE, d)

	a.Condition(func() (success bool) {
		return  regainStake6 < regainStake7
	})

	//modify system stamina to origin value
	tester.mdFreeStamina(t, d, sysFreeStamina)
	//free stamina regain should be different
	a.NoError(tester.sendTransferTrx(newAcct6, 1, transferAmount))
	a.NoError(tester.sendTransferTrx(newAcct7, 2 , transferAmount))
	a.NoError(d.ProduceBlocks(1))
	maxFreeStamina6,leftFreeStamina6,_,_ := getUserStamina(acctName6, d)
	maxFreeStamina7,leftFreeStamina7,_,_ := getUserStamina(acctName7, d)
	a.Equal(maxFreeStamina6, maxFreeStamina7)
	a.Condition(func() (success bool) {
		return leftFreeStamina6 > leftFreeStamina7
	})
	a.NoError(d.ProduceBlocks(newBlk))
	regainFree6 := getStaminaRegain(acctName6, FREE, d)
	regainFree7 := getStaminaRegain(acctName7, FREE, d)
	//account6's free stamina should regain slower than account7
	a.Condition(func() (success bool) {
		return regainFree6 < regainFree7
	})
	//tester.mdFreeStamina(t, d, sysFreeStamina)

}

//
// if user's total stamina is not enough to deduct trx net resource, trx will not be applied
//
func (tester *StakeTester) unableSendTrx(t *testing.T, d *Dandelion) {
	a := assert.New(t)
	acctName := "account8"
	tester.createNewAccount(t, d, acctName, 0)
	newAcct := d.Account(acctName)

	dgp := d.GlobalProps()
	sysFreeStamina := dgp.GetStaminaFree()
	var newFreeStamina uint64 = 100
	a.NotEqual(sysFreeStamina, newFreeStamina)
	tester.mdFreeStamina(t, d, newFreeStamina)

	receipt := newAcct.TrxReceipt(Transfer(acctName, tester.acc1.Name, 100, ""))
	a.Nil(receipt)

	maxFreeStamina,leftFreeStamina,maxStakeStamina,leftStakeStamina := getUserStamina(acctName, d)
	a.Equal(maxFreeStamina, leftFreeStamina)
	a.Equal(maxStakeStamina, leftStakeStamina)
	tester.mdFreeStamina(t, d, sysFreeStamina)

}


//
// if user's trx fail to apply, still deduct  user's stamina
//
func (tester *StakeTester) applyTrxFail(t *testing.T, d *Dandelion) {

	a := assert.New(t)
	acctName := "account9"
	tester.createNewAccount(t, d, acctName, 0)
	newAcct := d.Account(acctName)
	balance9 := newAcct.GetBalance().Value

	notExiAcct :=  "account10"
	a.False(d.Account(notExiAcct).CheckExist())
	receipt := newAcct.TrxReceipt(Transfer(acctName, notExiAcct, 10, ""))
	a.Equal(receipt.Status, prototype.StatusDeductStamina)
	consumeFree := newAcct.GetStaminaFree()
	consumeStake := newAcct.GetStamina()
	totalConsume := receipt.CpuUsage + receipt.NetUsage
	//apply trx fail , but stamina still be consumed
	a.Equal(totalConsume, consumeFree + consumeStake)
	a.Equal(balance9, d.Account(acctName).GetBalance().Value)

}

func (tester *StakeTester) createNewAccount(t *testing.T, d *Dandelion, name string, stakeAmount uint64) {
	a := assert.New(t)
    a.False(d.Account(name).CheckExist())
	priv, _ := prototype.GenerateNewKey()
	pub, _ := priv.PubKey()
	var (
		ops []*prototype.Operation
		transAmount uint64 = 10000 * constants.COSTokenDecimals
	)

	ops = append(ops, AccountCreate(constants.COSInitMiner, name, pub, 10, ""),
		         Transfer(constants.COSInitMiner, name, transAmount, ""))
	a.NoError(d.SendTrxByAccount(constants.COSInitMiner, ops...))
	a.NoError(d.ProduceBlocks(1))
	a.True(d.Account(name).CheckExist())
	d.PutAccount(name, priv)

	if stakeAmount > 0 {
		initminer := d.Account(constants.COSInitMiner)
		balanceInitMiner := initminer.GetBalance().Value
		a.NoError(d.SendTrxByAccount(initminer.Name, Stake(initminer.Name, name, stakeAmount)))
		a.NoError(d.ProduceBlocks(1))
		a.Equal(balanceInitMiner-stakeAmount, initminer.GetBalance().Value)
		a.Equal(stakeAmount, d.Account(name).GetStakeVest().Value)
	}

}

func (tester *StakeTester) mdFreeStamina(t *testing.T, d *Dandelion, freeStamina uint64) {
	a := assert.New(t)
	a.NoError(d.ModifyProps(func(oldProps *prototype.DynamicProperties) {
		oldProps.StaminaFree = freeStamina
	}))
	a.Equal(freeStamina, d.GlobalProps().GetStaminaFree())
}

func (tester *StakeTester) multipleTransfer(t *testing.T, d *Dandelion, sender string, opCnt int) uint64 {
	a := assert.New(t)
	var (
		consume uint64
		ops []*prototype.Operation
		transferAmount uint64 = 100
	)

	if opCnt > 3 {
		opCnt = 3
	}
	recMap := map[string]uint64{}
	balanceSender := d.Account(sender).GetBalance().Value
	for i := 0; i < opCnt; i++ {
		name := fmt.Sprintf("actor%d", i)
		recMap[name] = d.Account(name).GetBalance().Value
		ops = append(ops, Transfer(sender, name, transferAmount,""))
	}
	receipt := d.TrxReceiptByAccount(sender, ops...)
	a.NoError(checkError(receipt))
	for name,balance := range recMap {
		a.Equal(balance+transferAmount, d.Account(name).GetBalance().Value)
	}
	a.Equal(balanceSender-uint64(opCnt)*transferAmount, d.Account(sender).GetBalance().Value)
	consume = receipt.NetUsage + receipt.CpuUsage
	return consume
}

func (tester *StakeTester) sendTransferTrx(sender *DandelionAccount, opCnt int, transferAmount uint64) error {
	var ops []*prototype.Operation
	if opCnt > 3 {
		opCnt = 3
	}
	for i := 0; i < opCnt; i++ {
		name := fmt.Sprintf("actor%d", i)
		ops = append(ops, Transfer(sender.Name, name, transferAmount,""))
	}
	return sender.SendTrx(ops...)
}


func (tester *StakeTester) divideCeilBig(num,den *big.Int) *big.Int {
	tmp := new(big.Int)
	tmp.Div(num, den)
	if num.Mod(num, den).Uint64() > 0 {
		tmp.Add(tmp, big.NewInt(1))
	}
	return tmp
}

func getUserStamina(name string, d *Dandelion) (maxFree uint64,leftFree uint64, maxStake uint64, leftStake uint64)  {
	dgp := d.GlobalProps()
	acct := d.Account(name)
	maxFree = dgp.StaminaFree
    consumeFree := acct.GetStaminaFree()
	consumeStake := acct.GetStamina()

	rc := utils.ResourceLimiter{}

	_,leftFree = rc.GetFreeLeft(maxFree, consumeFree, acct.GetStaminaFreeUseBlock(), dgp.HeadBlockNumber)
	maxStake = d.CalculateUserMaxStamina(name)
	_,leftStake = rc.GetStakeLeft(consumeStake, acct.GetStaminaUseBlock(), dgp.HeadBlockNumber, maxStake)
	return
}

func getStaminaRegain(name string, st StaminaType, d *Dandelion) (regain uint64) {
	acct := d.Account(name)
	if st == FREE || st == STAKE {
		consumeFree := acct.GetStaminaFree()
		consumeStake := acct.GetStamina()
		maxFreeStamina,curFreeStamina,maxStakeStamina,curStakeStamina := getUserStamina(name, d)
		regain = consumeStake - (maxStakeStamina-curStakeStamina)
		if st == FREE {
			regain = consumeFree - (maxFreeStamina-curFreeStamina)
		}

	}
	return
}

func (tester *StakeTester) calNewConsume(consume uint64, interval uint64, period uint64) uint64 {
	blocks := big.NewInt(int64(period))
	precisionBig := big.NewInt(constants.LimitPrecision)
	oldConsumeBig := big.NewInt(int64(consume))
	oldConsumeBig.Mul(oldConsumeBig, precisionBig)

	avgOld := tester.divideCeilBig(oldConsumeBig, blocks)
	if interval < period {
		gap := big.NewInt(int64(blocks.Uint64() - interval))
		gap.Mul(gap, precisionBig)
		decay := tester.divideCeilBig(gap, blocks)

		avgOld.Mul(avgOld, decay)
		avgOld.Div(avgOld, precisionBig)
	} else {
		avgOld.SetUint64(0)
	}

	avgOld.Mul(avgOld, blocks).Div(avgOld, precisionBig)
	return avgOld.Uint64()
}
