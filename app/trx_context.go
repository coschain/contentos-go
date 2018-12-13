package app

import (
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm"
)


type TrxContext struct {
	vm.Injector
	Wrapper *prototype.TransactionWrapper
	db iservices.IDatabaseService
}

func NewTrxContext(wrapper *prototype.TransactionWrapper, db iservices.IDatabaseService) *TrxContext {
	return &TrxContext{ Wrapper:wrapper , db:db }
}

func (p *TrxContext) Verify()  {
	ownerGetter := func(name string) *prototype.Authority {
		account := &prototype.AccountName{Value: name}
		authWrap := table.NewSoAccountAuthorityObjectWrap(p.db, account)
		auth := authWrap.GetOwner()
		if auth == nil {
			panic("no owner auth")
		}
		return auth
	}

	tmpChainId := prototype.ChainId{Value: 0}
	p.verifyAuthority(tmpChainId, 2, ownerGetter)
}

func (p *TrxContext) verifyAuthority(cid prototype.ChainId, maxDepth uint32, owner AuthorityGetter) {
	pubs, err := p.Wrapper.SigTrx.ExportPubKeys(cid)
	if err != nil {
		panic(err)
	}
	verifyAuthority(p.Wrapper.SigTrx.Trx.Operations, pubs, maxDepth, owner)
}

func (p *TrxContext) RequireAuth(name string) error{
	return nil
}

func (p *TrxContext) Transfer(from, to string, amount uint64, memo string) error{
	return nil
}



func verifyAuthority(ops []*prototype.Operation, trxPubs []*prototype.PublicKeyType, max_recursion_depth uint32, owner AuthorityGetter) {
	//required_active := map[string]bool{}
	//required_posting := map[string]bool{}
	required_owner := map[string]bool{}
	other := []prototype.Authority{}

	for _, op := range ops {
		baseOp := getBaseOp(op)

		baseOp.GetAuthorities(&other)
		baseOp.GetRequiredOwner(&required_owner)
	}


	s := SignState{}
	s.Init(trxPubs, max_recursion_depth, owner)

	for k, _ := range required_owner {
		if !s.CheckAuthorityByName(k, 0, Owner) {
			panic("check owner authority failed")
		}
	}
}


func getBaseOp(op *prototype.Operation) prototype.BaseOperation {
	switch t := op.Op.(type) {
	case *prototype.Operation_Op1:
		return prototype.BaseOperation(t.Op1)
	case *prototype.Operation_Op2:
		return prototype.BaseOperation(t.Op2)
	case *prototype.Operation_Op3:
		return prototype.BaseOperation(t.Op3)
	case *prototype.Operation_Op4:
		return prototype.BaseOperation(t.Op4)
	case *prototype.Operation_Op5:
		return prototype.BaseOperation(t.Op5)
	case *prototype.Operation_Op6:
		return prototype.BaseOperation(t.Op6)
	case *prototype.Operation_Op7:
		return prototype.BaseOperation(t.Op7)
	case *prototype.Operation_Op8:
		return prototype.BaseOperation(t.Op8)
	case *prototype.Operation_Op9:
		return prototype.BaseOperation(t.Op9)
	case *prototype.Operation_Op10:
		return prototype.BaseOperation(t.Op10)
	default:
		return nil
	}
}