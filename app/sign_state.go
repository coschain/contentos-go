package app

import (
	"bytes"
	"github.com/coschain/contentos-go/proto/type-proto"
)

type authorityType uint16

const (
	posting authorityType = iota
	active  authorityType = iota
	owner   authorityType = iota
)

type SignState struct {
	// PublicKeyType can not use as key in map
	trxCarryedPubs []prototype.PublicKeyType
	approved       map[string]bool
	max_recursion  uint32
}

func (s *SignState) checkPub(key *prototype.PublicKeyType) bool {
	for _, k := range s.trxCarryedPubs {
		if bytes.Equal(key.Data, k.Data) {
			return true
		}
	}
	return false
}

func (s *SignState) CheckAuthority(name string, depth uint32, at authorityType) bool {
	// a speed up cache
	if _, ok := s.approved[name]; ok {
		return true
	}
	// a speed up cache
	auth, err := s.getAuthority(name, at)
	if err != nil {
		panic("")
	}

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
			if s.CheckAuthority(a.Name.Value, depth+1, at) {
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

func (s *SignState) getAuthority(name string, at authorityType) (*prototype.Authority, error) {
	// read Authority struct from DB
	switch at {
	case posting:
	case active:
	case owner:
	default:
	}
	return nil, nil
}
