package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// readPassword печатает prompt и считывает пароль с клавиатуры,
// не отображая вводимые символы на экране.
func readPassword(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(b), nil
}
