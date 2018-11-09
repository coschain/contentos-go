package types

import (
	"errors"
	"fmt"
	"io"

	"github.com/coschain/contentos-go/p2p/depend/common"
	"github.com/coschain/contentos-go/p2p/depend/common/serialization"
)

type MutableTransaction struct {
	Version  byte
	TxType   TransactionType
	Nonce    uint32
	GasPrice uint64
	GasLimit uint64
	Payer    common.Address
	Payload  Payload
	//Attributes []*TxAttribute
	attributes byte //this must be 0 now, Attribute Array length use VarUint encoding, so byte is enough for extension
	Sigs       []Sig
}

// output has no reference to self
func (self *MutableTransaction) IntoImmutable() (*Transaction, error) {
	sink := common.NewZeroCopySink(nil)
	err := self.serialize(sink)
	if err != nil {
		return nil, err
	}

	return TransactionFromRawBytes(sink.Bytes())
}

func (self *MutableTransaction) Hash() common.Uint256 {
	tx, err := self.IntoImmutable()
	if err != nil {
		return common.UINT256_EMPTY
	}
	return tx.Hash()
}

func (self *MutableTransaction) GetSignatureAddresses() []common.Address {
	address := make([]common.Address, 0, len(self.Sigs))
	for _, sig := range self.Sigs {
		m := int(sig.M)
		n := len(sig.PubKeys)

		if n == 1 {
			address = append(address, AddressFromPubKey(sig.PubKeys[0]))
		} else {
			addr, err := AddressFromMultiPubKeys(sig.PubKeys, m)
			if err != nil {
				return nil
			}
			address = append(address, addr)
		}
	}
	return address
}

// Serialize the Transaction
func (tx *MutableTransaction) serialize(sink *common.ZeroCopySink) error {
	err := tx.serializeUnsigned(sink)
	if err != nil {
		return err
	}

	sink.WriteVarUint(uint64(len(tx.Sigs)))
	for _, sig := range tx.Sigs {
		err = sig.Serialization(sink)
		if err != nil {
			return err
		}
	}

	return nil
}

func (tx *MutableTransaction) serializeUnsigned(sink *common.ZeroCopySink) error {
	sink.WriteByte(byte(tx.Version))
	sink.WriteByte(byte(tx.TxType))
	sink.WriteUint32(tx.Nonce)
	sink.WriteUint64(tx.GasPrice)
	sink.WriteUint64(tx.GasLimit)
	sink.WriteBytes(tx.Payer[:])

	//Payload
	if tx.Payload == nil {
		return errors.New("transaction payload is nil")
	}
	switch tx.Payload.(type) {
	//switch pl := tx.Payload.(type) {
	//case *payload.DeployCode:
	//	err := pl.Serialization(sink)
	//	if err != nil {
	//		return err
	//	}
	//case *payload.InvokeCode:
	//	err := pl.Serialization(sink)
	//	if err != nil {
	//		return err
	//	}
	default:
		return errors.New("wrong transaction payload type")
	}
	sink.WriteVarUint(uint64(tx.attributes))

	return nil
}

func (tx *MutableTransaction) DeserializeUnsigned(r io.Reader) error {
	var versiontype [2]byte
	_, err := io.ReadFull(r, versiontype[:])
	if err != nil {
		return err
	}
	nonce, err := serialization.ReadUint32(r)
	if err != nil {
		return err
	}
	gasPrice, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	gasLimit, err := serialization.ReadUint64(r)
	if err != nil {
		return err
	}
	tx.Version = versiontype[0]
	tx.TxType = TransactionType(versiontype[1])
	tx.Nonce = nonce
	tx.GasPrice = gasPrice
	tx.GasLimit = gasLimit
	if err := tx.Payer.Deserialize(r); err != nil {
		return err
	}

	switch tx.TxType {
	//case Invoke:
	//	tx.Payload = new(payload.InvokeCode)
	//case Deploy:
	//	tx.Payload = new(payload.DeployCode)
	default:
		return fmt.Errorf("unsupported tx type %v", tx.TxType)
	}

	err = tx.Payload.Deserialize(r)
	if err != nil {
		return err
	}

	//attributes
	length, err := serialization.ReadVarUint(r, 0)
	if err != nil {
		return err
	}
	if length != 0 {
		return fmt.Errorf("transaction attribute must be 0, got %d", length)
	}
	tx.attributes = 0

	return nil
}
