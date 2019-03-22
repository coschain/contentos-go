package app

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/injector"
)

type TrxContext struct {
	vminjector.Injector
	Wrapper     *prototype.EstimateTrxResult
	db          iservices.IDatabaseRW
	msg         []string
	recoverPubs []*prototype.PublicKeyType
	control     iservices.IGlobalPropRW
}

func NewTrxContext(wrapper *prototype.EstimateTrxResult, db iservices.IDatabaseRW, control *TrxPool) *TrxContext {
	return &TrxContext{Wrapper: wrapper, db: db, control: control }
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
	keyMaps := obtainKeyMap(p.Wrapper.SigTrx.Trx.Operations)
	verifyAuthority(keyMaps, p.recoverPubs, maxDepth, owner)
}

func (p *TrxContext) authGetter(name string) *prototype.Authority {
	account := &prototype.AccountName{Value: name}
	authWrap := table.NewSoAccountAuthorityObjectWrap(p.db, account)
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
	if balance < spent {
		panic(fmt.Sprintf("Endanger deduction Operation: %s, %d", caller, spent))
	}
	acc.MdBalance(&prototype.Coin{Value: balance - spent})
	return
}

// vm transfer just modify db data
func (p *TrxContext) TransferFromContractToUser(contract, owner, to string, amount uint64) {
	// need authority?
	c := table.NewSoContractWrap(p.db, &prototype.ContractId{Owner: &prototype.AccountName{Value: owner}, Cname: contract})
	balance := c.GetBalance().Value
	if balance < amount {
		panic(fmt.Sprintf("Endanger Transfer Operation: %s, %s, %s, %d", contract, owner, to, amount))
	}
	acc := table.NewSoAccountWrap(p.db, &prototype.AccountName{Value: to})
	// need atomic ?
	c.MdBalance(&prototype.Coin{Value: balance - amount})
	acc.MdBalance(&prototype.Coin{Value: acc.GetBalance().Value + amount})
	return
}

func (p *TrxContext) TransferFromUserToContract(from, contract, owner string, amount uint64) {
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

func (p *TrxContext) ContractCall(caller, fromOwner, fromContract, fromMethod, toOwner, toContract, toMethod string, params []byte, coins, maxGas uint64) {
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
		Gas: &prototype.Coin{ Value: maxGas },
	}
	eval := &InternalContractApplyEvaluator{ ctx: &ApplyContext{ db: p.db, trxCtx: p, control: p.control }, op: op }
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
			panic("check owner authority failed")
		}
	}
}
