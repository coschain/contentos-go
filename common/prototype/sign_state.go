package prototype

import (
	"bytes"
)

type AuthorityGetter func(string) *Authority

type AuthorityType uint16

const (
	Posting AuthorityType = iota
	Active  AuthorityType = iota
	Owner   AuthorityType = iota
)

type SignState struct {
	// PublicKeyType can not use as key in map
	trxCarryedPubs []*PublicKeyType
	approved       map[string]bool
	max_recursion  uint32
	PostingGetter AuthorityGetter
	ActiveGetter AuthorityGetter
	OwnerGetter AuthorityGetter
}

func (s *SignState) checkPub(key *PublicKeyType) bool {
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
	return s.CheckAuthority(auth,0, at)
}

func (s *SignState) CheckAuthority(auth *Authority, depth uint32, at AuthorityType) bool {

	var total_weight uint32 = 0
	for _, k := range auth.KeyAuths {
		if s.checkPub(k.Key) {
			total_weight += k.Weight
			if total_weight >= auth.WeightThreshold {
				return true
			}
		}
	}

	for _, a := range auth.AccountAuths {
		username := a.Name.Value
		if _, ok := s.approved[username]; !ok {
			if depth == s.max_recursion {
				continue
			}
			auth := s.getAuthority(username, at)
			if s.CheckAuthority(auth, depth+1, at) {
				s.approved[username] = true
				total_weight += a.Weight
				if total_weight >= auth.WeightThreshold {
					return true
				}
			}

		} else {
			total_weight += a.Weight
			if total_weight >= auth.WeightThreshold {
				return true
			}
		}
	}

	return total_weight >= auth.WeightThreshold
}

func (s *SignState) Init(pubs []*PublicKeyType,maxDepth uint32,posting AuthorityGetter,active AuthorityGetter,owner AuthorityGetter) {
	 copy(s.trxCarryedPubs,pubs)
	 s.max_recursion = maxDepth
	 s.PostingGetter = posting
	 s.ActiveGetter = active
	 s.OwnerGetter = owner
}

func (s *SignState) getAuthority(name string, at AuthorityType) *Authority {
	// read Authority struct from DB
	switch at {
	case Posting:
		return s.PostingGetter(name)
	case Active:
		return s.ActiveGetter(name)
	case Owner:
		return s.OwnerGetter(name)
	default:
	}
	return nil
}
