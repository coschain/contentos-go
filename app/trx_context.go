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
	Wrapper *prototype.TransactionWrapper
	db      iservices.IDatabaseService

	recoverPubs []*prototype.PublicKeyType
}

func NewTrxContext(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseService) *TrxContext {
	return &TrxContext{Wrapper: wrapper, db: db}
}

func (p *TrxContext) InitSigState(cid prototype.ChainId) error {
	pubs, err := p.Wrapper.SigTrx.ExportPubKeys(cid)
	if err != nil {
		return err
	}
	p.recoverPubs = append(p.recoverPubs, pubs...)
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

// vm transfer just modify db data
func (p *TrxContext) ContractTransfer(contract, owner, to string, amount uint64) error {
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
	return nil
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
			panic("check owner authority failed")
		}
	}
}
