// Package cli реализует интерактивный CLI-режим GophKeeper: пользователь
// один раз аутентифицируется, дальше выполняет команды в рамках одной
// непрерывной сессии без повторного ввода мастер-пароля.
package cli

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/client/grpcclient"
)

// state — состояние текущей сессии: подключение к серверу, токен
// авторизации и расшифрованный ключ шифрования данных (DEK).
// Живёт в памяти процесса от запуска до выхода из интерактивного режима.
type state struct {
	authClient   authv1.AuthServiceClient
	secretClient secretv1.SecretServiceClient
	token        string
	dek          []byte
	loggedIn     bool
}

// errExit — служебная "ошибка", которой сигнализируем о выходе из цикла.
var errExit = errors.New("exit")

// Run запускает интерактивный режим GophKeeper: подключается к серверу
// serverAddr и входит в цикл чтения команд из stdin.
func Run(serverAddr string) error {
	conn, err := grpcclient.Dial(serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	st := &state{
		authClient:   authv1.NewAuthServiceClient(conn),
		secretClient: secretv1.NewSecretServiceClient(conn),
	}

	fmt.Println("GophKeeper. Введите 'help' для списка команд.")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("gophkeeper> ")
		if !scanner.Scan() {
			fmt.Println()
			return nil
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		args := strings.Fields(line)
		cmd, rest := args[0], args[1:]

		if err := dispatch(st, cmd, rest); err != nil {
			if errors.Is(err, errExit) {
				return nil
			}
			fmt.Println("Ошибка:", err)
		}
	}
}
func dispatch(st *state, cmd string, args []string) error {
	switch cmd {
	case "register":
		return cmdRegister(st, args)
	case "login":
		return cmdLogin(st, args)
	case "list":
		return requireAuth(st, func() error { return cmdList(st) })
	case "get":
		return requireAuth(st, func() error { return cmdGet(st, args) })
	case "add":
		return requireAuth(st, func() error { return cmdAdd(st, args) })
	case "delete":
		return requireAuth(st, func() error { return cmdDelete(st, args) })
	case "help":
		printHelp()
		return nil
	case "exit", "quit":
		return errExit
	default:
		return fmt.Errorf("неизвестная команда %q, введите 'help'", cmd)
	}
}

func requireAuth(st *state, fn func() error) error {
	if !st.loggedIn {
		return errors.New("сначала выполните login или register")
	}
	return fn()
}

func printHelp() {
	fmt.Println(`Команды:
  register          зарегистрировать нового пользователя
  login             войти
  list              показать список секретов
  get <id>          показать расшифрованный секрет
  add               добавить новый секрет
  delete <id>       удалить секрет
  exit              выйти`)
}
