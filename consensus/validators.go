package consensus

import (
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/gobft/message"
	"time"
)

/********* implements gobft IPubValidator ***********/

type publicValidator struct {
	sab         *SABFT
	accountName string
}

func (sabft *SABFT) timeToNextSec() time.Duration {
	now := sabft.Ticker.Now()
	ceil := now.Add(time.Millisecond * 500).Round(time.Second)
	return ceil.Sub(now)
}

func (pv *publicValidator) VerifySig(digest, signature []byte) bool {
	// Warning: DO NOT remove the lock unless you know what you're doing
	pv.sab.RLock()
	defer pv.sab.RUnlock()

	return pv.verifySig(digest, signature)
}

func (pv *publicValidator) verifySig(digest, signature []byte) bool {
	acc := &prototype.AccountName{
		Value: pv.accountName,
	}
	return pv.sab.ctrl.VerifySig(acc, digest, signature)
}

func (pv *publicValidator) GetPubKey() message.PubKey {
	return message.PubKey(pv.accountName)
}

func (pv *publicValidator) GetVotingPower() int64 {
	return 1
}

func (pv *publicValidator) SetVotingPower(int64) {

}

/********* end gobft IPubValidator ***********/

/********* implements gobft IPrivValidator ***********/

type privateValidator struct {
	sab     *SABFT
	privKey *prototype.PrivateKeyType
	name    string
}

func (pv *privateValidator) Sign(digest []byte) []byte {
	// Warning: DO NOT remove the lock unless you know what you're doing
	pv.sab.RLock()
	defer pv.sab.RUnlock()

	return pv.sign(digest)
}

func (pv *privateValidator) sign(digest []byte) []byte {
	return pv.sab.ctrl.Sign(pv.privKey, digest)
}

func (pv *privateValidator) GetPubKey() message.PubKey {
	return message.PubKey(pv.name)
}

/********* end gobft IPrivValidator ***********/
