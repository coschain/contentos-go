package malnode

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/gogo/protobuf/proto"
	"time"
)

type MaliciousNode struct {
	*node.Node
}

func NewMaliciousNode(n *node.Node) *MaliciousNode {
	return &MaliciousNode{
		Node: n,
	}
}

func (n *MaliciousNode) Register(name string, constructor node.ServiceConstructor) error {
	return n.Node.Register(name, n.serviceProxyConstructor(name, constructor))
}

func (n *MaliciousNode) serviceProxyConstructor(name string, origin node.ServiceConstructor) (proxyConstructor node.ServiceConstructor) {
	proxyConstructor = origin
	switch name {
	case iservices.TxPoolServerName:
		proxyConstructor = func(ctx *node.ServiceContext) (node.Service, error) {
			svc, err := origin(ctx)
			if err != nil {
				return nil, err
			}
			s, ok := svc.(*app.TrxPool)
			if !ok {
				return nil, err
			}
			return NewTrxPoolServiceProxy(s, n), nil
		}
	}
	return
}

func (n *MaliciousNode) TrxPoolPrePushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) (newBlock *prototype.SignedBlock, newSkip prototype.SkipFlag, ok bool) {
	return blk, skip, true
}

func (n *MaliciousNode) TrxPoolPostPushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag, newBlock *prototype.SignedBlock, newSkip prototype.SkipFlag, ok bool, ret error) (newRet error) {
	return ret
}

func (n *MaliciousNode) TrxPoolPreGenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (newBpName string, newPre *prototype.Sha256, newTimestamp uint32, newPriKey *prototype.PrivateKeyType, newSkip prototype.SkipFlag, ok bool) {
	return bpName, pre, timestamp, priKey, skip, true
}

func (n *MaliciousNode) TrxPoolPostGenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag, newBpName string, newPre *prototype.Sha256, newTimestamp uint32, newPriKey *prototype.PrivateKeyType, newSkip prototype.SkipFlag, ok bool, retBlock *prototype.SignedBlock, retErr error) (newBlock *prototype.SignedBlock, newErr error) {
	if !ok {
		return nil, errors.New("malicious node refused to generate a block")
	}
	if retErr != nil {
		return retBlock, retErr
	}
	fmt.Printf("malicious node %s produced block: %d\n", bpName, retBlock.Id().BlockNum())

	data, _ := proto.Marshal(retBlock)
	block := new(prototype.SignedBlock)
	_ = proto.Unmarshal(data, block)

	var prevBlockId common.BlockID
	copy(prevBlockId.Data[:], pre.Hash)
	tx := &prototype.Transaction{
		RefBlockNum: common.TaposRefBlockNum(prevBlockId.BlockNum()),
		RefBlockPrefix: common.TaposRefBlockPrefix(prevBlockId.Data[:]),
		Expiration: &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix()) + 10},
	}
	tx.AddOperation(&prototype.TransferOperation{
		From:                 prototype.NewAccountName(bpName),
		To:                   prototype.NewAccountName("initminer"),
		Amount:               prototype.NewCoin(constants.COSInitSupply),
	})
	signTx := &prototype.SignedTransaction{Trx: tx}
	sig := signTx.Sign(priKey, prototype.ChainId{ Value:common.GetChainIdByName("main") })
	signTx.Signature = &prototype.SignatureType{Sig: sig}
	block.Transactions = append(block.Transactions, &prototype.TransactionWrapper{
		SigTrx:               signTx,
		Receipt:              &prototype.TransactionReceipt{
			Status:               prototype.StatusSuccess,
			NetUsage:             123,
			CpuUsage:             456,
		},
	})
	id := block.CalculateMerkleRoot()
	block.SignedHeader.Header.TransactionMerkleRoot = &prototype.Sha256{Hash: id.Data[:]}
	block.SignedHeader.BlockProducerSignature = new(prototype.SignatureType)
	_ = block.SignedHeader.Sign(priKey)

	return block, retErr
}
