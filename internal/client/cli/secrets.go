package cli

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	secretv1 "github.com/SatzhanDev/gophKeeper/api/proto/secret/v1"
	"github.com/SatzhanDev/gophKeeper/internal/client/crypto"
	"github.com/SatzhanDev/gophKeeper/internal/client/grpcclient"
)

// cmdList реализует команду "list": выводит список секретов пользователя
// без расшифровки (metadata хранится открытым текстом).
func cmdList(st *state) error {
	ctx := grpcclient.AuthContext(context.Background(), st.token)
	resp, err := st.secretClient.ListSecrets(ctx, &secretv1.ListSecretsRequest{})
	if err != nil {
		return err
	}

	for _, s := range resp.GetSecrets() {
		fmt.Printf("%d\t%s\t%s\n", s.GetId(), s.GetType(), s.GetMetadata())
	}
	return nil
}

// cmdGet реализует команду "get <id>": запрашивает секрет с сервера
// и расшифровывает его локально ключом DEK текущей сессии.
func cmdGet(st *state, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: get <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("некорректный id: %w", err)
	}

	ctx := grpcclient.AuthContext(context.Background(), st.token)
	resp, err := st.secretClient.GetSecret(ctx, &secretv1.GetSecretRequest{Id: id})
	if err != nil {
		return err
	}

	plaintext, err := crypto.Decrypt(st.dek, resp.GetSecret().GetData())
	if err != nil {
		return fmt.Errorf("не удалось расшифровать данные")
	}
	fmt.Println(string(plaintext))
	return nil
}

// cmdAdd реализует команду "add <type> <metadata> <данные>": шифрует
// данные ключом DEK текущей сессии и сохраняет секрет на сервере.
func cmdAdd(st *state, args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("использование: add <type> <metadata> <данные>")
	}

	secretType, ok := parseSecretType(args[0])
	if !ok {
		return fmt.Errorf("неизвестный тип: %s (login|text|binary|card)", args[0])
	}
	metadata := args[1]
	data := strings.Join(args[2:], " ")

	encrypted, err := crypto.Encrypt(st.dek, []byte(data))
	if err != nil {
		return err
	}

	ctx := grpcclient.AuthContext(context.Background(), st.token)
	resp, err := st.secretClient.CreateSecret(ctx, &secretv1.CreateSecretRequest{
		Type:     secretType,
		Data:     encrypted,
		Metadata: metadata,
	})
	if err != nil {
		return err
	}

	fmt.Printf("Секрет создан, id=%d\n", resp.GetId())
	return nil
}

// cmdDelete реализует команду "delete <id>": удаляет секрет на сервере.
func cmdDelete(st *state, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("использование: delete <id>")
	}
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("некорректный id: %w", err)
	}

	ctx := grpcclient.AuthContext(context.Background(), st.token)
	_, err = st.secretClient.DeleteSecret(ctx, &secretv1.DeleteSecretRequest{Id: id})
	if err != nil {
		return err
	}

	fmt.Println("Секрет удалён")
	return nil
}

// parseSecretType переводит текстовое имя типа из команды пользователя
// в protobuf-перечисление SecretType.
func parseSecretType(s string) (secretv1.SecretType, bool) {
	switch s {
	case "login":
		return secretv1.SecretType_SECRET_TYPE_LOGIN_PASSWORD, true
	case "text":
		return secretv1.SecretType_SECRET_TYPE_TEXT, true
	case "binary":
		return secretv1.SecretType_SECRET_TYPE_BINARY, true
	case "card":
		return secretv1.SecretType_SECRET_TYPE_CARD, true
	default:
		return 0, false
	}
}
