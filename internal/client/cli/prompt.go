package cli

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// readPassword печатает prompt и считывает пароль с клавиатуры,
// не отображая вводимые символы на экране.
//
// Объявлена как переменная-функция (а не обычная func), а не потому что
// это чем-то лучше по производительности — а чтобы в юнит-тестах команд
// register/login можно было подменить её на фейковую реализацию,
// не трогая реальный терминал.
var readPassword = func(prompt string) (string, error) {
	fmt.Print(prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", err
	}
	return string(b), nil
}
