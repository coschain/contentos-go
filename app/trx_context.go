package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/injector"
	"github.com/coschain/contentos-go/common/constants"
)

type TrxContext struct {
	vminjector.Injector
	DynamicGlobalPropsRW
	Wrapper         *prototype.TransactionWrapper
	msg         []string
	recoverPubs []*prototype.PublicKeyType
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

func (p *TrxContext) GetVmRemainCpuStamina(name string) uint64 {
	return p.control.GetAllRemainStamina(name) - (p.netMap[name].raw * constants.NetConsumePointNum / constants.NetConsumePointDen)
}

func (p *TrxContext) CheckNet(db iservices.IDatabaseRW, sizeInBytes uint64) {
	keyMaps := p.Wrapper.SigTrx.GetOpCreatorsMap()
	netUse := sizeInBytes * uint64(float64(constants.NetConsumePointNum)/float64(constants.NetConsumePointDen))
	for name := range keyMaps {
		p.netMap[name] = &resourceUnit{}
		freeLeft := p.resourceLimiter.GetFreeLeft(db, name, p.control.GetProps().HeadBlockNumber)
		stakeLeft := p.resourceLimiter.GetStakeLeft(db, name, p.control.GetProps().HeadBlockNumber)
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
		now := p.control.GetProps().HeadBlockNumber

		var paid uint64 = 0

		if !p.resourceLimiter.ConsumeFree(db, caller, staminaUse, now) {
			paid += p.resourceLimiter.GetFreeLeft(db, caller, now)
			p.resourceLimiter.ConsumeFreeLeft(db, caller, now)

		} else {
			paid = staminaUse
			// free resource already enough
			m[caller].realCost = paid

			continue

		}

		left := staminaUse - paid

		if !p.resourceLimiter.Consume(db, caller, left, now) {
			// never failed ?
			paid += p.resourceLimiter.GetStakeLeft(db, caller, now)
			p.resourceLimiter.ConsumeLeft(db, caller, now)

		} else {
			paid += left

		}
		m[caller].realCost = paid
	}
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

func NewTrxContext(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseRW,control *TrxPool) *TrxContext {
	return &TrxContext{
		DynamicGlobalPropsRW: DynamicGlobalPropsRW{ db:db },
		Wrapper: wrapper,
		control: control,
		gasMap: make(map[string]*resourceUnit),
		netMap: make(map[string]*resourceUnit),
		resourceLimiter: control.resourceLimiter,
	}
}

func NewTrxContextWithSigningKey(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseRW, key *prototype.PublicKeyType, control *TrxPool, observer iservices.ITrxObserver) *TrxContext {
	return &TrxContext{
		DynamicGlobalPropsRW: DynamicGlobalPropsRW{ db:db },
		Wrapper: wrapper,
		recoverPubs: []*prototype.PublicKeyType{ key },
		observer: observer,
		gasMap: make(map[string]*resourceUnit),
		netMap: make(map[string]*resourceUnit),
		resourceLimiter: control.resourceLimiter,
		control:control,
	}
}

func (p *TrxContext) InitSigState(cid prototype.ChainId) error {
	pub, err := p.Wrapper.SigTrx.ExportPubKeys(cid)
	if err != nil {
		return err
	}
	p.recoverPubs = append(p.recoverPubs, pub)
	return nil
}

func (p *TrxContext) VerifySignature() {
	p.verifyAuthority(2, p.authGetter)
}

func (p *TrxContext) verifyAuthority(maxDepth uint32, owner AuthorityGetter) {
	//keyMaps := obtainKeyMap(p.Wrapper.SigTrx.Trx.Operations)
	keyMaps := p.Wrapper.SigTrx.GetOpCreatorsMap()
	if len(keyMaps) != 1 {
		panic("trx creator is not unique")
	}
	verifyAuthority(keyMaps, p.recoverPubs, maxDepth, owner)
}

func (p *TrxContext) authGetter(name string) *prototype.PublicKeyType {
	account := &prototype.AccountName{Value: name}
	authWrap := table.NewSoAccountWrap(p.db, account)
	auth := authWrap.GetOwner()
	if auth == nil {
		panic("no owner auth")
	}
	return auth
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
	keyMaps := map[string]bool{}
	keyMaps[name] = true

	defer func() {
		if ret := recover(); ret != nil {
			err = errors.New(fmt.Sprint(ret))
		}
	}()

	verifyAuthority(keyMaps, p.recoverPubs, 2, p.authGetter)
	return nil
}

func (p *TrxContext) DeductGasFee(caller string, spent uint64) {
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: caller})
	balance := acc.GetBalance().Value
	if balance < spent {
		panic(fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent))
	}
	acc.Md(func(tInfo *table.SoAccount) {
		tInfo.Balance = &prototype.Coin{Value: balance - spent}
	})
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

	c.Md(func(tInfo *table.SoContract) {
		tInfo.Balance = &prototype.Coin{Value: balance - amount}
	})

	acc.Md(func(tInfo *table.SoAccount) {
		tInfo.Balance = &prototype.Coin{Value: acc.GetBalance().Value + amount}
	})
	return
}

func (p *TrxContext) TransferFromUserToContract(from, contract, owner string, amount uint64) {
	opAssert(false, "function not opened")
	p.RequireAuth( from )

	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: from})
	balance := acc.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, from, amount))
	}
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	c.Md(func(tInfo *table.SoContract) {
		tInfo.Balance = &prototype.Coin{Value: balance + amount}
	})
	acc.Md(func(tInfo *table.SoAccount) {
		tInfo.Balance = &prototype.Coin{Value: balance - amount}
	})
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
	from.Md(func(tInfo *table.SoContract) {
		tInfo.Balance = &prototype.Coin{Value: fromBalance - amount}
	})
	to.Md(func(tInfo *table.SoContract) {
		tInfo.Balance = &prototype.Coin{Value: toBalance + amount}
	})
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

func verifyAuthority(keyMaps map[string]bool, trxPubs []*prototype.PublicKeyType, max_recursion_depth uint32, owner AuthorityGetter) {
	//required_active := map[string]bool{}
	//required_posting := map[string]bool{}
	//other := []prototype.Authority{}

	s := SignState{}
	s.Init(trxPubs, max_recursion_depth, owner)

	for k := range keyMaps {
		if !s.CheckAuthorityByName(k, 0, Owner) {
			panic("check owner authority failed")
		}
	}
}
