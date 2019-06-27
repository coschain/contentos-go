package common

import (
	"hash/crc32"
)

const (
	ChainIdMainNet uint32 = iota
	ChainIdTestNet
	ChainIdDevNet

	BuiltinChainIdCount
)

var sKnownChains = map[string]uint32 {
	"main": ChainIdMainNet,
	"test": ChainIdTestNet,
	"dev": ChainIdDevNet,
}

func GetChainIdByName(name string) uint32 {
	chainId, builtin := sKnownChains[name]
	if !builtin {
		chainId = crc32.ChecksumIEEE([]byte(name))
	}
	return chainId
}
