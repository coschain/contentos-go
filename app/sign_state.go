package app

import (
	"bytes"
	"github.com/coschain/contentos-go/prototype"
)

type AuthorityGetter func(string) *prototype.Authority

type AuthorityType uint16

const (
	//Posting AuthorityType = iota
	//Active  AuthorityType = iota
	Owner   AuthorityType = iota
)

type SignState struct {
	// PublicKeyType can not use as key in map
	trxCarryedPubs []*prototype.PublicKeyType
	approved       map[string]bool
	max_recursion  uint32
	//PostingGetter  AuthorityGetter
	//ActiveGetter   AuthorityGetter
	OwnerGetter    AuthorityGetter
}

func (s *SignState) checkPub(key *prototype.PublicKeyType) bool {
	for _, k := range s.trxCarryedPubs {
		if bytes.Equal(key.Data, k.Data) {
			return true
		}
	}
	return false
}

func (s *SignState) CheckAuthorityByName(name string, depth uint32, at AuthorityType) bool {
	// a speed up cache
	if _, ok := s.approved[name]; ok {
		return true
	}
	// a speed up cache
	auth := s.getAuthority(name, at)
	return s.CheckAuthority(auth, 0, at)
}

func (s *SignState) CheckAuthority(auth *prototype.Authority, depth uint32, at AuthorityType) bool {
	if s.checkPub(auth.Key) {
		return true
	} else {
		return false
	}
}

func (s *SignState) Init(pubs []*prototype.PublicKeyType, maxDepth uint32, owner AuthorityGetter) {
	s.trxCarryedPubs = s.trxCarryedPubs[:0]
	s.trxCarryedPubs = append(s.trxCarryedPubs, pubs...)
	s.max_recursion = maxDepth
	//s.PostingGetter = posting
	//s.ActiveGetter = active
	s.OwnerGetter = owner
}

func (s *SignState) getAuthority(name string, at AuthorityType) *prototype.Authority {
	// read Authority struct from DB
	switch at {
	//case Posting:
	//	return s.PostingGetter(name)
	//case Active:
	//	return s.ActiveGetter(name)
	case Owner:
		return s.OwnerGetter(name)
	default:
	}
	return nil
}
