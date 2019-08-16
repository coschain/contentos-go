package prototype

import (
	"errors"
	"github.com/coschain/contentos-go/common/constants"
)

var (
	sErrLength = errors.New("invalid length")
	sErrCharset = errors.New("invalid char")
)

func ValidAccountName(s string) error {
	if len(s) < constants.MinAccountNameLength || len(s) > constants.MaxAccountNameLength {
		return sErrLength
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z') {
			return sErrCharset
		}
	}
	return nil
}

func ValidVarName(s string) error {
	size := len(s)
	if size < 1 || size > 64 {
		return sErrLength
	}
	c := rune(s[0])
	if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_') {
		return sErrCharset
	}
	if size > 1 {
		for _, c = range s[1:] {
			if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c == '_') {
				return sErrCharset
			}
		}
	}
	return nil
}

func stringLengthValidator(s string, min, max int) error {
	if len(s) < min || len(s) > max {
		return sErrLength
	}
	return nil
}

var ValidContractName = ValidVarName
var ValidContractMethodName = ValidVarName
var ValidContractTableName = ValidVarName
var AtMost1KChars = func(s string) error { return stringLengthValidator(s, 0, 1024 * 1) }
var AtMost4KChars = func(s string) error { return stringLengthValidator(s, 0, 1024 * 4) }
