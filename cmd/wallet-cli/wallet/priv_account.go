package wallet

// As a expediency until further implement

type Account struct {
	Name   string
	PubKey string
}

type PrivAccount struct {
	Account
	PrivKey string
	Expire  int64
}

type EncryptAccount struct {
	Account
	Cipher     string // a cipher algorithm from aes
	CipherText string // encrypted privkey
	Iv         string // the iv
	Version    uint8  // version of format
}
