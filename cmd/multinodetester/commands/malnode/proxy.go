package malnode

import (
	"github.com/coschain/contentos-go/app"
	"github.com/coschain/contentos-go/prototype"
)

type ITrxPoolServiceProxyCallback interface {
	TrxPoolPrePushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) (newBlock *prototype.SignedBlock, newSkip prototype.SkipFlag, ok bool)
	TrxPoolPostPushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag, newBlock *prototype.SignedBlock, newSkip prototype.SkipFlag, ok bool, ret error) (newRet error)
	TrxPoolPreGenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (newBpName string, newPre *prototype.Sha256, newTimestamp uint32, newPriKey *prototype.PrivateKeyType, newSkip prototype.SkipFlag, ok bool)
	TrxPoolPostGenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag, newBpName string, newPre *prototype.Sha256, newTimestamp uint32, newPriKey *prototype.PrivateKeyType, newSkip prototype.SkipFlag, ok bool, retBlock *prototype.SignedBlock, retErr error) (newBlock *prototype.SignedBlock, newErr error)
}

type TrxPoolServiceProxy struct {
	*app.TrxPool
	callback ITrxPoolServiceProxyCallback
}

func NewTrxPoolServiceProxy(s *app.TrxPool, callback ITrxPoolServiceProxyCallback) *TrxPoolServiceProxy {
	return &TrxPoolServiceProxy {
		TrxPool: s,
		callback: callback,
	}
}

func (proxy *TrxPoolServiceProxy) PushBlock(blk *prototype.SignedBlock, skip prototype.SkipFlag) error {
	if newBlock, newSkip, ok := proxy.callback.TrxPoolPrePushBlock(blk, skip); ok {
		ret := proxy.TrxPool.PushBlock(newBlock, newSkip)
		return proxy.callback.TrxPoolPostPushBlock(blk, skip, newBlock, newSkip, true, ret)
	} else {
		return proxy.callback.TrxPoolPostPushBlock(blk, skip, newBlock, newSkip, false, nil)
	}
}

func (proxy *TrxPoolServiceProxy) GenerateAndApplyBlock(bpName string, pre *prototype.Sha256, timestamp uint32, priKey *prototype.PrivateKeyType, skip prototype.SkipFlag) (*prototype.SignedBlock, error) {
	if newBpName, newPre, newTimestamp, newPriKey, newSkip, ok := proxy.callback.TrxPoolPreGenerateAndApplyBlock(bpName, pre, timestamp, priKey, skip); ok {
		blk, err := proxy.TrxPool.GenerateAndApplyBlock(newBpName, newPre, newTimestamp, newPriKey, newSkip)
		return proxy.callback.TrxPoolPostGenerateAndApplyBlock(bpName, pre, timestamp, priKey, skip, newBpName, newPre, newTimestamp, newPriKey, newSkip, true, blk, err)
	} else {
		return proxy.callback.TrxPoolPostGenerateAndApplyBlock(bpName, pre, timestamp, priKey, skip, newBpName, newPre, newTimestamp, newPriKey, newSkip, true, nil, nil)
	}
}
