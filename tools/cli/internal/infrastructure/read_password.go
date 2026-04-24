package infrastructure

import (
	"fmt"
	"syscall"

	"golang.org/x/term"
)

func ReadPassword() (string, error) {
	fmt.Print("Password: ")
	passwordBytes, err := term.ReadPassword(syscall.Stdin)
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	return string(passwordBytes), nil
}
