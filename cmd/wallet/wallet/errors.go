package wallet

import "fmt"

type UnknownLockedAccountError struct {
	Name string
}

func (e *UnknownLockedAccountError) Error() string {
	return fmt.Sprintf("unknown locked account: %v", e.Name)
}

type ReentrantUnlockedAccountError struct {
	Name string
}

func (e *ReentrantUnlockedAccountError) Error() string {
	return fmt.Sprintf("re entrant unlocked account: %v", e.Name)
}

type AccountNotFound struct {
	LocalName string
}

func (e AccountNotFound) Error() string {
	return fmt.Sprintf("%s is not found in wallet try to load or create", e.LocalName)
}
