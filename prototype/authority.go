package prototype

import "github.com/pkg/errors"

func (this *Authority) add_authority(k *PublicKeyType, w uint32) {

}

func NewAuthorityFromPubKey(pubKey *PublicKeyType) *Authority {
	m := &Authority{
		WeightThreshold: 1,
		KeyAuths: []*KvKeyAuth{
			{
				Key:    pubKey,
				Weight: 1,
			},
		},
	}
	return m
}

func (m *Authority) Validate() error {
	if m == nil {
		return errors.New("npe")
	}

	//TODO check valid

	return nil
}
