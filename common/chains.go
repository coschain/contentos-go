package common

import (
	"hash/crc32"
)

func GetChainIdByName(name string) uint32 {
	return crc32.ChecksumIEEE([]byte(name))
}

const (
	ChainNameMainNet = "main"
	ChainNameTestNet = "test"
	ChainNameDevNet = "dev"
)

var (
	ChainIdMainNet = GetChainIdByName(ChainNameMainNet)
	ChainIdTestNet = GetChainIdByName(ChainNameTestNet)
	ChainIdDevNet  = GetChainIdByName(ChainNameDevNet)
)
