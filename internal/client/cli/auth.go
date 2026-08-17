package cli

import (
	"context"
	"crypto/rand"
	"fmt"

	authv1 "github.com/SatzhanDev/gophKeeper/api/proto/auth/v1"
	"github.com/SatzhanDev/gophKeeper/internal/client/crypto"
)

// cmdRegister реализует команду "register <login>": генерирует новый DEK,
// оборачивает его ключом, выведенным из мастер-пароля, и регистрирует
// пользователя на сервере.
func cmdRegister(st *state, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: register <login>")
	}
	login := args[0]

	password, err := readPassword("Мастер-пароль: ")
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	params := crypto.DefaultKDFParams()
	kek := crypto.DeriveKey(password, salt, params)

	dek, err := crypto.GenerateDEK()
	if err != nil {
		return err
	}
	wrappedDEK, err := crypto.WrapDEK(dek, kek)
	if err != nil {
		return err
	}

	// Сообщение собирается через сеттеры (Opaque API, режим Hybrid — см.
	// Makefile), а не через struct literal вида &authv1.RegisterRequest{Login: login}.
	// Поля пока формально ещё экспортируемые (Hybrid — переходная ступень
	// перед полным Opaque, нужна для совместимости с protoc-gen-grpc-gateway),
	// но напрямую их использовать больше не следует — цель миграции
	// (полный Opaque) сделает прямой доступ невозможным.
	req := &authv1.RegisterRequest{}
	req.SetLogin(login)
	req.SetPassword(password)
	req.SetKdfSalt(salt)
	req.SetKdfTime(params.Time)
	req.SetKdfMemoryKb(params.MemoryKB)
	req.SetKdfThreads(uint32(params.Threads))
	req.SetWrappedDek(wrappedDEK)

	resp, err := st.authClient.Register(context.Background(), req)
	if err != nil {
		return err
	}

	st.token = resp.GetToken()
	st.dek = dek
	st.loggedIn = true
	fmt.Println("Регистрация успешна")
	return nil
}

// cmdLogin реализует команду "login <login>": аутентифицирует пользователя
// и восстанавливает DEK из мастер-пароля и данных, полученных от сервера.
func cmdLogin(st *state, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: login <login>")
	}
	login := args[0]

	password, err := readPassword("Мастер-пароль: ")
	if err != nil {
		return err
	}

	req := &authv1.LoginRequest{}
	req.SetLogin(login)
	req.SetPassword(password)

	resp, err := st.authClient.Login(context.Background(), req)
	if err != nil {
		return err
	}

	params := crypto.KDFParams{
		Time:     resp.GetKdfTime(),
		MemoryKB: resp.GetKdfMemoryKb(),
		Threads:  uint8(resp.GetKdfThreads()),
	}
	kek := crypto.DeriveKey(password, resp.GetKdfSalt(), params)
	dek, err := crypto.UnwrapDEK(resp.GetWrappedDek(), kek)
	if err != nil {
		return fmt.Errorf("неверный мастер-пароль")
	}

	st.token = resp.GetToken()
	st.dek = dek
	st.loggedIn = true
	fmt.Println("Вход выполнен")
	return nil
}
