package utils

import (
	"golang.org/x/crypto/ssh/terminal"
)

type MyPasswordReader struct{}

func (pr MyPasswordReader) ReadPassword(fd int) ([]byte, error) {
	return terminal.ReadPassword(fd)
}
