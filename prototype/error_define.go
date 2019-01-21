package prototype

import "github.com/pkg/errors"

var (
	ErrNpe          = errors.New("Null Pointer")
	ErrKeyLength    = errors.New("Key Length Error")
	ErrHashLength   = errors.New("Hash Length Error")
	ErrSigLength    = errors.New("Signature Length Error")
	ErrCoinOverflow = errors.New("Coin Overflow")
	ErrVestOverflow = errors.New("Vest Overflow")
	ErrPubKeyFormatErr = errors.New("Public Key Format Error")

)

type Exception struct {
	ErrorType int
	HelpString string
	ErrorString string
}

func (e *Exception) ToString() string  {
	return e.HelpString + e.ErrorString
}

const StatusSuccess  = 200
const StatusError  = 500

// trx error
const StatusErrorTrxId = 100000
const StatusErrorTrxPriKeyToPubKey = 100001
const StatusErrorTrxExportPubKey = 100002
const StatusErrorTrxOverflow = 100003
const StatusErrorTrxTaPos = 100004
const StatusErrorTrxBlockHeaderCheck = 100005
const StatusErrorTrxSize = 100006
const StatusErrorTrxClearPending = 100007
const StatusErrorTrxPubKeyCmp = 100008
const StatusErrorTrxMaxBlockSize = 100009
const StatusErrorTrxDuplicateCheck = 100010
const StatusErrorTrxExpire = 100011
const StatusErrorTrxMerkleCheck = 100012
const StatusErrorTrxApplyInvoice = 100013
const StatusErrorTrxMaxUndo = 100013

// Db error
const StatusErrorDbEndTrx = 200000
const StatusErrorDbTruncate = 200001
const StatusErrorDbCreate = 200002
const StatusErrorDbTag = 200003
const StatusErrorDbUpdate = 200004
const StatusErrorDbExist = 200005
const StatusErrorDbDelete = 200006

// op error
const StatusErrorOp = 300000
const StatusErrorOpWithVmRun = 300001
const StatusErrorVmOp = 300002