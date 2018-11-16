package prototype

import "github.com/pkg/errors"

func (this *Authority) add_authority(k *PublicKeyType, w uint32) {

}


func (m *Authority) Validate() error {
	if m == nil {
		return errors.New("npe")
	}

	//TODO check valid

	return nil
}
