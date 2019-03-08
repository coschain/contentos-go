package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/utils"
	"github.com/coschain/contentos-go/vm/injector"
)

type resourceUnit struct {
	raw	uint64 // may be net in byte or cpu gas
	realCost uint64 // real cost resource
}

type TrxContext struct {
	vminjector.Injector
	Wrapper         *prototype.TransactionWrapper
	db              iservices.IDatabaseService
	msg             []string
	recoverPubs     []*prototype.PublicKeyType
	control         *TrxPool
	gasMap          map[string]*resourceUnit
	netMap          map[string]*resourceUnit
	resourceLimiter utils.IResourceLimiter
}

func NewTrxContext(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseService, control *TrxPool) *TrxContext {
	return &TrxContext{Wrapper: wrapper, db: db, control: control, gasMap: make(map[string]*resourceUnit), netMap: make(map[string]*resourceUnit), resourceLimiter: control.resourceLimiter}
}

func (p *TrxContext) InitSigState(cid prototype.ChainId) error {
	pub, err := p.Wrapper.SigTrx.ExportPubKeys(cid)
	if err != nil {
		return err
	}
	p.recoverPubs = append(p.recoverPubs, pub)
	return nil
}

func (p *TrxContext) GetVmRemainCpuStamina(name string) uint64 {
	return p.control.GetAllRemainStamina(name) - (p.netMap[name].raw * constants.NetConsumePointNum/constants.NetConsumePointDen)
}

func (p *TrxContext) CheckNet(sizeInBytes uint64) {
	keyMaps := obtainKeyMap(p.Wrapper.SigTrx.Trx.Operations)
	netUse := sizeInBytes * uint64(float64(constants.NetConsumePointNum)/float64(constants.NetConsumePointDen))
	for name := range keyMaps {
		p.netMap[name] = &resourceUnit{}
		freeLeft := p.resourceLimiter.GetFreeLeft(name, p.control.GetProps().HeadBlockNumber)
		stakeLeft := p.resourceLimiter.GetStakeLeft(name, p.control.GetProps().HeadBlockNumber)
		if freeLeft >= netUse {
			p.netMap[name].raw = sizeInBytes
			continue
		} else {
			if stakeLeft >= netUse-freeLeft {
				p.netMap[name].raw = sizeInBytes
				continue
			} else {
				p.netMap = make(map[string]*resourceUnit)
				mustSuccess(false, "net resource not enough", prototype.StatusError)
			}
		}
	}
}

func (p *TrxContext) GetNetUse() uint64 {
	all := uint64(0)
	for _,use := range p.netMap {
		all += use.realCost
	}
	return all
}

func (p *TrxContext) GetCpuUse() uint64 {
	all := uint64(0)
	for _,use := range p.gasMap {
		all += use.realCost
	}
	return all
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

func (p *TrxContext) Log(msg string) {
	fmt.Print(msg)
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

func (p *TrxContext) deductStamina(m map[string]*resourceUnit,num,den uint64) {
	rate := float64(num)/float64(den)
	for caller, spent := range m {
		staminaUse := uint64(float64(spent.raw) * rate)
		now := p.control.GetProps().HeadBlockNumber
		var paid uint64 = 0
		if !p.resourceLimiter.ConsumeFree(caller, staminaUse, now) {
			paid += p.resourceLimiter.GetFreeLeft(caller,now)
			p.resourceLimiter.ConsumeFreeLeft(caller,now)
		} else {
			paid = staminaUse
			// free resource already enough
			m[caller].realCost = paid
			continue
		}

		left := staminaUse - paid
		if !p.resourceLimiter.Consume(caller, left, now) {
			// never failed ?
			paid += p.resourceLimiter.GetStakeLeft(caller, now)
			p.resourceLimiter.ConsumeLeft(caller, now)
		} else {
			paid += left
		}
		m[caller].realCost = paid
	}
}

func (p *TrxContext) DeductAllNet() {
	p.deductStamina(p.netMap,constants.NetConsumePointNum,constants.NetConsumePointDen)
}

func (p *TrxContext) DeductAllCpu() {
	p.deductStamina(p.gasMap,constants.CpuConsumePointNum,constants.CpuConsumePointDen)
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
	if !p.control.ctx.Config().ResourceCheck {
		return
	}
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

func (p *TrxContext) ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, remainGas uint64) {
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
	}
	eval := &InternalContractApplyEvaluator{ctx: &ApplyContext{db: p.db, trxCtx: p, control: p.control}, op: op,remainGas:remainGas}
	eval.Apply()
}

func obtainKeyMap(ops []*prototype.Operation) map[string]bool {
	keyMaps := map[string]bool{}
	for _, op := range ops {
		baseOp := prototype.GetBaseOperation(op)

		//baseOp.GetAuthorities(&other)
		baseOp.GetSigner(&keyMaps)
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
