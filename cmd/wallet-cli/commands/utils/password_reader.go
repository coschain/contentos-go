package utils

type PasswordReader interface {
	ReadPassword(fd int) ([]byte, error)
}
