package prototype

import "github.com/pkg/errors"

var (
	ErrNpe 			= errors.New("Null Pointer")
	ErrKeyLength 	= errors.New("Key Length Error")
	ErrHashLength 	= errors.New("Hash Length Error")
	ErrSigLength 	= errors.New("Signature Length Error")

)
