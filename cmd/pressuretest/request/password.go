package request

type MyMockPasswordReader struct{}

func (pr MyMockPasswordReader) ReadPassword(fd int) ([]byte, error) {
	return []byte{1,2,3}, nil
}