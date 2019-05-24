package app

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/injector"
)

type TrxContext struct {
	vminjector.Injector
	DynamicGlobalPropsRW
	Wrapper         *prototype.TransactionWrapper
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

func (p *TrxContext) getRemainStakeStamina(db iservices.IDatabaseRW,name string) uint64 {
	wraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wraper.GetProps()
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	maxStamina := p.calculateUserMaxStamina(db,name)
	return p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), gp.HeadBlockNumber, maxStamina)
}

func (p *TrxContext) getRemainFreeStamina(db iservices.IDatabaseRW,name string) uint64 {
	wraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	gp := wraper.GetProps()
	accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
	return p.resourceLimiter.GetFreeLeft(gp.GetStaminaFree(),accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), gp.HeadBlockNumber)
}

func (p *TrxContext) GetVmRemainCpuStamina(name string) uint64 {
	allRemain := p.getRemainStakeStamina(p.db,name) + p.getRemainFreeStamina(p.db,name)
	return allRemain - (p.netMap[name].raw * constants.NetConsumePointNum / constants.NetConsumePointDen)
}

func (p *TrxContext) CheckNet(db iservices.IDatabaseRW, sizeInBytes uint64) {
	keyMaps := p.Wrapper.SigTrx.GetOpCreatorsMap()
	netUse := sizeInBytes * uint64(float64(constants.NetConsumePointNum)/float64(constants.NetConsumePointDen))
	dgpWraper := table.NewSoGlobalWrap(db, &constants.GlobalId)
	for name := range keyMaps {
		p.netMap[name] = &resourceUnit{}

		accountWrap := table.NewSoAccountWrap(db, &prototype.AccountName{Value:name})
		maxStamina := p.calculateUserMaxStamina(db,name)
		freeLeft := p.resourceLimiter.GetFreeLeft(dgpWraper.GetProps().GetStaminaFree(),accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), dgpWraper.GetProps().HeadBlockNumber)
		stakeLeft := p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), dgpWraper.GetProps().HeadBlockNumber,maxStamina)
		if freeLeft >= netUse {
			p.netMap[name].raw = sizeInBytes
			continue
		} else {
			if stakeLeft >= netUse-freeLeft {
				p.netMap[name].raw = sizeInBytes
				continue
			} else {
				errInfo := fmt.Sprintf("net resource not enough, user:%v, have:%v, need:%v",name,freeLeft+stakeLeft,netUse)
				opAssert(false, errInfo)
			}
		}
	}
}

func (p *TrxContext) deductStamina(db iservices.IDatabaseRW,m map[string]*resourceUnit, num, den uint64) {
	rate := float64(num) / float64(den)

	for caller, spent := range m {
		staminaUse := uint64(float64(spent.raw) * rate)
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
			paid += p.resourceLimiter.GetFreeLeft(freeStaminaMaxByBp,accountWrap.GetStaminaFree(), accountWrap.GetStaminaFreeUseBlock(), now)

			accountWrap.MdStaminaFree(freeStaminaMaxByBp)
			accountWrap.MdStaminaFreeUseBlock(now)

		} else {
			accountWrap.MdStaminaFree(newFreeStamina)
			accountWrap.MdStaminaFreeUseBlock(now)
			paid = staminaUse
			// free resource already enough
			m[caller].realCost = paid

			continue

		}

		left := staminaUse - paid

		maxStamina := p.calculateUserMaxStamina(db,caller)
		if ok,newStamina := p.resourceLimiter.Consume(accountWrap.GetStamina(), left, accountWrap.GetStaminaUseBlock(), now,maxStamina);!ok {
			// never failed ?
			paid += p.resourceLimiter.GetStakeLeft(accountWrap.GetStamina(), accountWrap.GetStaminaUseBlock(), now, maxStamina)

			accountWrap.MdStamina(maxStamina)
			accountWrap.MdStaminaUseBlock(now)
		} else {
			accountWrap.MdStamina(newStamina)
			accountWrap.MdStaminaUseBlock(now)
			paid += left
		}
		m[caller].realCost = paid
	}
}

func (p *TrxContext) calculateUserMaxStamina(db iservices.IDatabaseRW,name string) uint64 {
	return p.control.calculateUserMaxStamina(db,name)
}

func (p *TrxContext) DeductAllNet(db iservices.IDatabaseRW) {
	p.deductStamina(db, p.netMap, constants.NetConsumePointNum, constants.NetConsumePointDen)
}

func (p *TrxContext) DeductAllCpu(db iservices.IDatabaseRW) {
	p.deductStamina(db, p.gasMap, constants.CpuConsumePointNum, constants.CpuConsumePointDen)
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

func (p *TrxContext) RecordGasFee(caller string, spent uint64) {
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

func NewTrxContext(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseRW, signer string, control *TrxPool, observer iservices.ITrxObserver) *TrxContext {
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

func (p *TrxContext) DeductGasFee(caller string, spent uint64) {
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: caller})
	balance := acc.GetBalance().Value
	if balance < spent {
		panic(fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent))
	}
	acc.MdBalance(&prototype.Coin{Value: balance - spent})
	return
}

// vm transfer just modify db data
func (p *TrxContext) TransferFromContractToUser(contract, owner, to string, amount uint64) {
	opAssert(false, "function not opened")
	// TODO need authority

	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	balance := c.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, to, amount))
	}
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: to})

	c.MdBalance(&prototype.Coin{Value: balance - amount})
	acc.MdBalance(&prototype.Coin{Value: acc.GetBalance().Value + amount})
	return
}

func (p *TrxContext) TransferFromUserToContract(from, contract, owner string, amount uint64) {
	opAssert(false, "function not opened")
	opAssertE(p.RequireAuth( from ), fmt.Sprintf("requireAuth('%s') failed", from))

	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: from})
	balance := acc.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, from, amount))
	}
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	c.MdBalance(&prototype.Coin{Value: balance + amount})
	acc.MdBalance(&prototype.Coin{Value: balance - amount})
	return
}

func (p *TrxContext) TransferFromContractToContract(fromContract, fromOwner, toContract, toOwner string, amount uint64) {
	opAssert(false, "function not opened")
	// TODO checkAuth

	from := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: fromOwner}, Cname: fromContract})
	to := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: toOwner}, Cname: toContract})
	fromBalance := from.GetBalance().Value
	if fromBalance < amount {
		panic(fmt.Sprintf("Insufficient balance of contract: %s.%s, %d < %d", fromOwner, fromContract, fromBalance, amount))
	}
	toBalance := to.GetBalance().Value
	from.MdBalance(&prototype.Coin{Value: fromBalance - amount})
	to.MdBalance(&prototype.Coin{Value: toBalance + amount})
}

func (p *TrxContext) ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, remainGas uint64) {
	opAssert(false, "function not opened")
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
	eval := &InternalContractApplyEvaluator{ctx: &ApplyContext{db: p.db, vmInjector: p, control: p.control, log:p.control.log}, op: op, remainGas: remainGas}
	eval.Apply()
}
