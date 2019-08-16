package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/cache"
	"github.com/coschain/contentos-go/vm/injector"
	"github.com/go-interpreter/wagon/exec"
	"github.com/sirupsen/logrus"
)

type TrxContext struct {
	vminjector.Injector
	DynamicGlobalPropsRW
	Wrapper         *prototype.TransactionWrapperWithInfo
	msg         []string
	signer      string
	observer iservices.ITrxObserver
	output *prototype.OperationReceiptWithInfo
	control         *TrxPool
	gasMap          map[string]*resourceUnit
	netMap          map[string]*resourceUnit
	resourceLimiter utils.IResourceLimiter
}

type resourceUnit struct {
	raw      uint64 // may be net in byte or cpu gas
	realCost uint64 // real cost resource
}

func (p *TrxContext) getRemainStakeStamina(db iservices.IDatabaseRW,name string) (string,uint64) {
	wraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wraper.GetProps()
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	maxStamina := p.calculateUserMaxStamina(db,name)
	return p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), gp.HeadBlockNumber, maxStamina)
}

func (p *TrxContext) getRemainFreeStamina(db iservices.IDatabaseRW,name string) (string,uint64) {
	wraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wraper.GetProps()
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	return p.resourceLimiter.GetFreeLeft(gp.GetStaminaFree(),accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), gp.HeadBlockNumber)
}

func (p *TrxContext) GetVmRemainCpuStamina(name string) uint64 {
	_,stakeRemain := p.getRemainStakeStamina(p.db,name)
	_,freeRemain := p.getRemainFreeStamina(p.db,name)
	allRemain := stakeRemain + freeRemain
	return allRemain - (p.netMap[name].raw * constants.NetConsumePointNum / constants.NetConsumePointDen)
}

func (p *TrxContext) CheckNet(db iservices.IDatabaseRW, sizeInBytes uint64) {
	keyMaps := p.Wrapper.SigTrx.GetOpCreatorsMap()
	netUse := sizeInBytes * constants.NetConsumePointNum
	dgpWraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	for name := range keyMaps {
		p.netMap[name] = &resourceUnit{}

		accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
		maxStamina := p.calculateUserMaxStamina(db,name)
		freeOver,freeLeft := p.resourceLimiter.GetFreeLeft(dgpWraper.GetProps().GetStaminaFree(),accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), dgpWraper.GetProps().HeadBlockNumber)
		stakeOver,stakeLeft := p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), dgpWraper.GetProps().HeadBlockNumber,maxStamina)

		if freeLeft >= netUse {
			p.netMap[name].raw = sizeInBytes
			continue
		} else {
			if stakeLeft >= netUse-freeLeft {
				p.netMap[name].raw = sizeInBytes
				continue
			} else {
				if freeOver == constants.FreeStaminaOverFlow || stakeOver == constants.StakeStaminaOverFlow {
					p.RecordStaminaFee(name,0) // a fake record to let this trx into block, then we can update user's stamina
				}
				errInfo := fmt.Sprintf("net resource not enough, user:%v, have:%v, need:%v",name,freeLeft+stakeLeft,netUse)
				opAssert(false, errInfo)
			}
		}
	}
}

func (p *TrxContext) deductStamina(db iservices.IDatabaseRW,m map[string]*resourceUnit, isNet bool) {
	for caller, spent := range m {
		var staminaUse uint64 = 0
		if isNet {
			staminaUse = spent.raw * constants.NetConsumePointNum
		} else {
			staminaUse = spent.raw / constants.CpuConsumePointDen
		}
		dgpWraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
		now := dgpWraper.GetProps().HeadBlockNumber

		var paid uint64 = 0

		accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:caller})
		if !accountWrap.CheckExist() {
			// todo other choice ?
			continue
		}
		freeStaminaMaxByBp := dgpWraper.GetProps().GetStaminaFree()
		if ok,newFreeStamina := p.resourceLimiter.ConsumeFree(freeStaminaMaxByBp,accountWrap.GetStaminaFree(), staminaUse,accountWrap.GetStaminaFreeUseBlock(), now);!ok {
			_,freeLeft := p.resourceLimiter.GetFreeLeft(freeStaminaMaxByBp,accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), now)

			paid += freeLeft
			accountWrap.SetStaminaFree(freeStaminaMaxByBp)
			accountWrap.SetStaminaFreeUseBlock(now)
		} else {
			accountWrap.SetStaminaFree(newFreeStamina)
			accountWrap.SetStaminaFreeUseBlock(now)
			paid = staminaUse
			// free resource already enough
			m[caller].realCost = paid

			continue
		}

		left := staminaUse - paid

		maxStamina := p.calculateUserMaxStamina(db,caller)
		if ok,newStamina := p.resourceLimiter.Consume(accountWrap.GetStamina(), left, accountWrap.GetStaminaUseBlock(), now,maxStamina);!ok {
			// never failed ?
			_,stakeLeft := p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), now, maxStamina)
			paid += stakeLeft
			accountWrap.SetStamina(maxStamina)
			accountWrap.SetStaminaUseBlock(now)
		} else {
			accountWrap.SetStamina(newStamina)
			accountWrap.SetStaminaUseBlock(now)
			paid += left
		}
		m[caller].realCost = paid
	}
}

func (p *TrxContext) calculateUserMaxStamina(db iservices.IDatabaseRW,name string) uint64 {
	return p.control.calculateUserMaxStamina(db,name)
}

func (p *TrxContext) DeductAllNet(db iservices.IDatabaseRW) {
	p.deductStamina(db, p.netMap, true)
}

func (p *TrxContext) DeductAllCpu(db iservices.IDatabaseRW) {
	p.deductStamina(db, p.gasMap, false)
}

func (p *TrxContext) Finalize() {
	p.setUsage()
}
func (p *TrxContext) SetStatus(s uint32) {
	p.Wrapper.Receipt.Status = s
}
func (p *TrxContext) setUsage() {
	p.Wrapper.Receipt.NetUsage = p.GetNetUse()
	p.Wrapper.Receipt.CpuUsage = p.GetCpuUse()
}

func (p *TrxContext) RecordOperationStaminaFee(){
	p.RecordStaminaFee( p.signer, constants.CommonOpStamina )
}

func (p *TrxContext) RecordStaminaFee(caller string, spent uint64) {
//	if !p.control.ctx.Config().ResourceCheck {
//		return
//	}
	// if same caller call multi times
	if v, ok := p.gasMap[caller]; ok {
		newSpent := v.raw + spent
		p.gasMap[caller].raw = newSpent
	} else {
		p.gasMap[caller] = &resourceUnit{}
		p.gasMap[caller].raw = spent
	}
}

func (p *TrxContext) HasGasFee() bool {
	return len(p.gasMap) > 0
}

func (p *TrxContext) GetNetUse() uint64 {
	all := uint64(0)
	for _, use := range p.netMap {
		all += use.realCost
	}
	return all
}

func (p *TrxContext) GetCpuUse() uint64 {
	all := uint64(0)
	for _, use := range p.gasMap {
		all += use.realCost
	}
	return all
}

func NewTrxContext(wrapper *prototype.TransactionWrapperWithInfo, db iservices.IDatabaseRW, signer string, control *TrxPool, observer iservices.ITrxObserver) *TrxContext {
	return &TrxContext{
		DynamicGlobalPropsRW: DynamicGlobalPropsRW{ db:db },
		Wrapper: wrapper,
		signer: signer,
		observer: observer,
		gasMap: make(map[string]*resourceUnit),
		netMap: make(map[string]*resourceUnit),
		resourceLimiter: control.resourceLimiter,
		control:control,
	}
}

func (p *TrxContext) Error(code uint32, msg string) {
	p.Wrapper.Receipt.ErrorInfo = msg
	//p.Wrapper.Receipt.Status = 500
}

func (p *TrxContext) StartNextOp() {
	p.output = &prototype.OperationReceiptWithInfo{VmConsole: ""}
	p.Wrapper.Receipt.OpResults = append(p.Wrapper.Receipt.OpResults, p.output)
}

func (p *TrxContext) Log(msg string) {
	p.output.VmConsole += msg
}

func (p *TrxContext) RequireAuth(name string) (err error) {
	if name != p.signer {
		return fmt.Errorf("requireAuth('%s') failed, signed by '%s'", name, p.signer)
	}
	return nil
}

func (p *TrxContext) DeductStamina(caller string, spent uint64) {
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: caller})
	balance := acc.GetBalance().Value
	if balance < spent {
		panic(fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent))
	}
	acc.SetBalance(&prototype.Coin{Value: balance - spent})
	return
}

// vm transfer just modify db data
func (p *TrxContext) TransferFromContractToUser(contract, owner, to string, amount uint64) {

	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	balance := c.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, to, amount))
	}
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: to})

	c.SetBalance(&prototype.Coin{Value: c.GetBalance().Value - amount})
	acc.SetBalance(&prototype.Coin{Value: acc.GetBalance().Value + amount})
	return
}

func (p *TrxContext) TransferFromUserToContract(from, contract, owner string, amount uint64) {
	//opAssert(false, "function not opened")
	opAssertE(p.RequireAuth( from ), fmt.Sprintf("requireAuth('%s') failed", from))

	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: from})
	balance := acc.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, from, amount))
	}
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	c.SetBalance(&prototype.Coin{Value: c.GetBalance().Value + amount})
	acc.SetBalance(&prototype.Coin{Value: acc.GetBalance().Value - amount})
	return
}

func (p *TrxContext) TransferFromContractToContract(fromContract, fromOwner, toContract, toOwner string, amount uint64) {
	if fromContract == toContract && fromOwner == toOwner {
		return
	}
	from := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: fromOwner}, Cname: fromContract})
	to := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: toOwner}, Cname: toContract})
	fromBalance := from.GetBalance().Value
	if fromBalance < amount {
		panic(fmt.Sprintf("Insufficient balance of contract: %s.%s, %d < %d", fromOwner, fromContract, fromBalance, amount))
	}
	toBalance := to.GetBalance().Value
	from.SetBalance(&prototype.Coin{Value: fromBalance - amount})
	to.SetBalance(&prototype.Coin{Value: toBalance + amount})
}

func (p *TrxContext) ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, remainGas uint64, preVm *exec.VM) {

	op := &prototype.InternalContractApplyOperation{
		FromCaller: &prototype.AccountName{ Value: caller },
		FromOwner: &prototype.AccountName{ Value: fromOwner },
		FromContract: fromContract,
		FromMethod: fromMethod,
		ToOwner: &prototype.AccountName{ Value: toOwner },
		ToContract: toContract,
		ToMethod: toMethod,
		Params: params,
		Amount: &prototype.Coin{ Value: coins },
	}
	eval := &InternalContractApplyEvaluator{BaseDelegate: BaseDelegate{delegate:p}, op: op, remainGas: remainGas, preVm:preVm}
	eval.Apply()
}

func (p *TrxContext) ContractABI(owner, contract string) string {
	cid := &prototype.ContractId {
		Owner: prototype.NewAccountName(owner),
		Cname: contract,
	}
	return table.NewSoContractWrap(p.db, cid).GetAbi()
}

func (p *TrxContext) GetBlockProducers() (names []string) {
	nameList := table.SBlockProducerOwnerWrap{Dba:p.db}
	_ = nameList.ForEachByOrder(nil, nil, nil, nil, func(mVal *prototype.AccountName, sVal *prototype.AccountName, idx uint32) bool {
		if table.NewSoBlockProducerWrap(p.db, mVal).GetBpVest().Active {
			names = append(names, mVal.Value)
		}
		return true
	})
	return
}

func (p *TrxContext) DiscardAccountCache(name string) {
	p.control.DiscardAccountCache(name)
}

//

//
// implements ApplyDelegate interface
//

func (p *TrxContext) Database() iservices.IDatabaseRW {
	return p.db
}

func (p *TrxContext) GlobalProp() iservices.IGlobalPropRW {
	return p
}

func (p *TrxContext) VMInjector() vminjector.Injector {
	return p
}

func (p *TrxContext) TrxObserver() iservices.ITrxObserver {
	return p.observer
}

func (p *TrxContext) Logger() *logrus.Logger {
	return p.control.log
}

func (p *TrxContext) VmCache() *vmcache.VmCache {
	return p.control.vmCache
}
