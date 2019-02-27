package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/injector"
)

const (
	netConsumePoint = 10
	cpuConsumePoint = 1
)

type TrxContext struct {
	vminjector.Injector
	Wrapper         *prototype.EstimateTrxResult
	db              iservices.IDatabaseService
	msg             []string
	recoverPubs     []*prototype.PublicKeyType
	control         *TrxPool
	gasMap          map[string]uint64
	netMap          map[string]uint64
	resourceLimiter utils.IResourceLimiter
}

func NewTrxContext(wrapper *prototype.EstimateTrxResult, db iservices.IDatabaseService, control *TrxPool) *TrxContext {
	return &TrxContext{Wrapper: wrapper, db: db, control: control, gasMap: make(map[string]uint64), netMap: make(map[string]uint64), resourceLimiter: utils.IResourceLimiter(utils.NewResourceLimiter(db))}
}

func (p *TrxContext) InitSigState(cid prototype.ChainId) error {
	pubs, err := p.Wrapper.SigTrx.ExportPubKeys(cid)
	if err != nil {
		return err
	}
	p.recoverPubs = append(p.recoverPubs, pubs...)
	return nil
}

func (p *TrxContext) CheckNet(sizeInBytes uint64) {
	keyMaps := obtainKeyMap(p.Wrapper.SigTrx.Trx.Operations)
	netUse := sizeInBytes * netConsumePoint
	for name := range keyMaps {
		freeLeft := p.resourceLimiter.GetFreeLeft(name, p.control.GetProps().HeadBlockNumber)
		stakeLeft := p.resourceLimiter.GetStakeLeft(name, p.control.GetProps().HeadBlockNumber)
		if freeLeft >= netUse {
			p.netMap[name] = sizeInBytes
			continue
		} else {
			if stakeLeft >= netUse-freeLeft {
				p.netMap[name] = sizeInBytes
				continue
			} else {
				p.netMap = make(map[string]uint64)
				mustSuccess(false, "net resource not enough", prototype.StatusError)
			}
		}
	}
}

func (p *TrxContext) VerifySignature() {
	p.verifyAuthority(2, p.authGetter)
}

func (p *TrxContext) verifyAuthority(maxDepth uint32, owner AuthorityGetter) {
	keyMaps := obtainKeyMap(p.Wrapper.SigTrx.Trx.Operations)
	verifyAuthority(keyMaps, p.recoverPubs, maxDepth, owner)
}

func (p *TrxContext) authGetter(name string) *prototype.Authority {
	account := &prototype.AccountName{Value: name}
	authWrap := table.NewSoAccountAuthorityObjectWrap(p.db, account)
	auth := authWrap.GetOwner()
	if auth == nil {
		mustSuccess(false, "no owner auth", prototype.StatusErrorDbExist)
	}
	return auth
}

func (p *TrxContext) Error(code uint32, msg string) {
	p.Wrapper.Receipt.ErrorInfo = msg
	//p.Wrapper.Receipt.Status = 500
}

func (p *TrxContext) AddOpReceipt(code uint32, gas uint64, msg string) {
	r := &prototype.OperationReceiptWithInfo{Status: code, GasUsage: gas, VmConsole: msg}
	p.Wrapper.Receipt.OpResults = append(p.Wrapper.Receipt.OpResults, r)
}

func (p *TrxContext) Log(msg string) {
	fmt.Print(msg)
	p.Wrapper.Receipt.OpResults = append(p.Wrapper.Receipt.OpResults, &prototype.OperationReceiptWithInfo{VmConsole: msg})
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
	mustSuccess(spent <= balance, fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent), prototype.StatusErrorTrxValueCompare)
	acc.MdBalance(&prototype.Coin{Value: balance - spent})
}

func (p *TrxContext) DeductAllGasFee() bool {

	useGas := false
	for caller, spent := range p.gasMap {
		acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: caller})
		balance := acc.GetBalance().Value
		mustSuccess(spent <= balance, fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent), prototype.StatusErrorTrxValueCompare)
		acc.MdBalance(&prototype.Coin{Value: balance - spent})
		useGas = true
	}
	return useGas
}

func (p *TrxContext) DeductAllNet() {
	for caller, spent := range p.netMap {
		netUse := spent * netConsumePoint

		if !p.resourceLimiter.ConsumeFree(caller, netUse, p.control.GetProps().HeadBlockNumber) {
			p.resourceLimiter.ConsumeFreeLeft(caller, p.control.GetProps().HeadBlockNumber)
		} else {
			// free resource already enough
			continue
		}

		if !p.resourceLimiter.Consume(caller, netUse, p.control.GetProps().HeadBlockNumber) {
			p.resourceLimiter.ConsumeLeft(caller, p.control.GetProps().HeadBlockNumber)
		}
	}
}

func (p *TrxContext) DeductAllCpu() bool {
	useGas := false
	for caller, spent := range p.gasMap {
		cpuUse := spent * cpuConsumePoint

		if !p.resourceLimiter.ConsumeFree(caller, cpuUse, p.control.GetProps().HeadBlockNumber) {
			p.resourceLimiter.ConsumeFreeLeft(caller, p.control.GetProps().HeadBlockNumber)
		} else {
			// free resource already enough
			continue
		}

		if !p.resourceLimiter.Consume(caller, cpuUse, p.control.GetProps().HeadBlockNumber) {
			// never failed ?
			p.resourceLimiter.ConsumeLeft(caller, p.control.GetProps().HeadBlockNumber)
		}
		useGas = true
	}
	return useGas
}

func (p *TrxContext) RecordGasFee(caller string, spent uint64) {
	// if same caller call multi times
	if v, ok := p.gasMap[caller]; ok {
		newSpent := v + spent
		p.gasMap[caller] = newSpent
	} else {
		p.gasMap[caller] = spent
	}
}

func (p *TrxContext) HasGasFee() bool {
	return len(p.gasMap) > 0
}

// vm transfer just modify db data
func (p *TrxContext) TransferFromContractToUser(contract, owner, to string, amount uint64) {
	// need authority?
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	balance := c.GetBalance().Value
	mustSuccess(balance >= amount, fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, to, amount), prototype.StatusErrorTrxPubKeyCmp)
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: to})
	// need atomic ?
	c.MdBalance(&prototype.Coin{Value: balance - amount})
	acc.MdBalance(&prototype.Coin{Value: acc.GetBalance().Value + amount})
	return
}

func (p *TrxContext) TransferFromUserToContract(from, contract, owner string, amount uint64) {
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: from})
	balance := acc.GetBalance().Value
	mustSuccess(balance >= amount, fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, from, amount), prototype.StatusErrorTrxPubKeyCmp)
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	c.MdBalance(&prototype.Coin{Value: balance + amount})
	acc.MdBalance(&prototype.Coin{Value: balance - amount})
	return
}

func (p *TrxContext) TransferFromContractToContract(fromContract, fromOwner, toContract, toOwner string, amount uint64) {
	from := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: fromOwner}, Cname: fromContract})
	to := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: toOwner}, Cname: toContract})
	fromBalance := from.GetBalance().Value
	mustSuccess(fromBalance >= amount, fmt.Sprintf("Insufficient balance of contract: %s.%s, %d < %d", fromOwner, fromContract, fromBalance, amount), prototype.StatusErrorTrxPubKeyCmp)
	toBalance := to.GetBalance().Value
	from.MdBalance(&prototype.Coin{Value: fromBalance - amount})
	to.MdBalance(&prototype.Coin{Value: toBalance + amount})
}

func (p *TrxContext) ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, maxGas uint64) {
	op := &prototype.InternalContractApplyOperation{
		FromCaller:   &prototype.AccountName{Value: caller},
		FromOwner:    &prototype.AccountName{Value: fromOwner},
		FromContract: fromContract,
		FromMethod:   fromMethod,
		ToOwner:      &prototype.AccountName{Value: toOwner},
		ToContract:   toContract,
		ToMethod:     toMethod,
		Params:       params,
		Amount:       &prototype.Coin{Value: coins},
		Gas:          &prototype.Coin{Value: maxGas},
	}
	eval := &InternalContractApplyEvaluator{ctx: &ApplyContext{db: p.db, trxCtx: p, control: p.control}, op: op}
	eval.Apply()
}

func obtainKeyMap(ops []*prototype.Operation) map[string]bool {
	keyMaps := map[string]bool{}
	for _, op := range ops {
		baseOp := prototype.GetBaseOperation(op)

		//baseOp.GetAuthorities(&other)
		baseOp.GetRequiredOwner(&keyMaps)
	}
	return keyMaps
}

func verifyAuthority(keyMaps map[string]bool, trxPubs []*prototype.PublicKeyType, max_recursion_depth uint32, owner AuthorityGetter) {
	//required_active := map[string]bool{}
	//required_posting := map[string]bool{}
	//other := []prototype.Authority{}

	s := SignState{}
	s.Init(trxPubs, max_recursion_depth, owner)

	for k := range keyMaps {
		if !s.CheckAuthorityByName(k, 0, Owner) {
			mustSuccess(false, "check owner authority failed", prototype.StatusErrorTrxVerifyAuth)
		}
	}
}
