package wallet

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBaseHDWallet_GenerateFromMnemonic(t *testing.T) {
	a := assert.New(t)
	wallet := NewBaseHDWallet("1", "")
	pubkeystr, privkeystr, _ := wallet.GenerateFromMnemonic("situate icon cluster install same burst vanish exchange tiny radar tourist labor exercise palm slab parrot drum spy liberty face flower hammer use walk")
	a.Equal("COS66qMKdw7Khcyhvr5FWGQEMhn5XksqL1wxXoXYyftj4pKbF1ku5", pubkeystr)
	a.Equal("3qBXa1xEzyzppu2BWnuJEFTW5HUFnXnJ3D9GiY636PNJSeXxrk", privkeystr)
}
