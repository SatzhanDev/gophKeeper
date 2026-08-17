.PHONY: build build-server build-client build-client-all build-client-linux build-client-windows build-client-darwin run-server run-client test test-integration vet fmt cover proto migrate-up migrate-down migrate-create certs

BIN_DIR := bin
PROTO_FILES := $(wildcard api/proto/*/v1/*.proto)

# Дефолты для локального запуска. ?= означает "присвоить, только если
# переменная ещё не задана" — если ты уже экспортировал DATABASE_DSN
# в своём шелле, make возьмёт твоё значение, а не эти дефолты.
# Приоритет: `make run-server DATABASE_DSN=...` (аргумент команды) >
# экспортированная переменная окружения > дефолт из этой строки.
DATABASE_DSN ?= postgres://postgres:postgres@localhost:5433/gophkeeper?sslmode=disable
JWT_SECRET   ?= dev-secret-change-me
GRPC_PORT    ?= 50051
HTTP_PORT    ?= 8080
SERVER_ADDR  ?= localhost:50051
TLS_CERT_FILE    ?= certs/server.crt
TLS_KEY_FILE     ?= certs/server.key
TLS_CA_CERT_FILE ?= certs/server.crt


VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X 'github.com/SatzhanDev/gophKeeper/internal/pkg/version.Version=$(VERSION)' \
              -X 'github.com/SatzhanDev/gophKeeper/internal/pkg/version.BuildDate=$(BUILD_DATE)'

build-server:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-server ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-client ./cmd/client

## Собрать server и client в bin/
build: build-server build-client

## Кросс-сборка клиента под все три требуемые по ТЗ платформы разом.
build-client-all: build-client-linux build-client-windows build-client-darwin

## Клиент под Linux (amd64)
build-client-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-client-linux-amd64 ./cmd/client

## Клиент под Windows (amd64). Расширение .exe обязательно — иначе Windows
## не опознает файл как исполняемый.
build-client-windows:
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-client-windows-amd64.exe ./cmd/client

## Клиент под macOS: amd64 (Intel) и arm64 (Apple Silicon) отдельными
## бинарниками — Go не умеет собирать один "универсальный" файл под обе
## архитектуры без дополнительных внешних инструментов (lipo).
build-client-darwin:
	GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-client-darwin-amd64 ./cmd/client
	GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/gophkeeper-client-darwin-arm64 ./cmd/client

## Сгенерировать самоподписанный TLS-сертификат для локального запуска
## (CN=localhost, потому что клиент по умолчанию ходит на localhost).
## Приватный ключ и сертификат не коммитятся в git (см. .gitignore) —
## каждый разработчик генерирует свою пару локально одной командой.
certs:
	mkdir -p certs
	openssl req -x509 -newkey rsa:4096 -keyout $(TLS_KEY_FILE) -out $(TLS_CERT_FILE) \
	        -days 365 -nodes -subj "/CN=localhost"

## Запустить сервер/клиент без сборки бинарника
run-server:
	DATABASE_DSN=$(DATABASE_DSN) JWT_SECRET=$(JWT_SECRET) GRPC_PORT=$(GRPC_PORT) HTTP_PORT=$(HTTP_PORT) \
	TLS_CERT_FILE=$(TLS_CERT_FILE) TLS_KEY_FILE=$(TLS_KEY_FILE) \
	go run ./cmd/server

run-client:
	GOPHKEEPER_SERVER=$(SERVER_ADDR) TLS_CA_CERT_FILE=$(TLS_CA_CERT_FILE) go run ./cmd/client

## Накатить все непримененные миграции на базу из DATABASE_DSN
migrate-up:
	migrate -database "$(DATABASE_DSN)" -path migrations up

## Откатить последнюю миграцию
migrate-down:
	migrate -database "$(DATABASE_DSN)" -path migrations down 1

## Создать пару файлов новой миграции: make migrate-create name=add_foo
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

## Пакеты, для которых считаем покрытие: без сгенерированного
## protobuf-кода (api/proto/... — там нет и не должно быть собственных
## тестов, это механическая (де)сериализация) и без cmd/... (тонкие точки
## входа main(), которые только связывают уже протестированные компоненты).
COVER_PKGS := $(shell go list ./... | grep -v '/api/proto/' | grep -v '/cmd/')

## Юнит-тесты со сбором покрытия в coverage.out. Интеграционные тесты
## (storage/postgres/integration_test.go) собраны под build-тегом
## "integration" и в этот прогон не попадают — им нужен Docker.
test:
	go test $(COVER_PKGS) -race -coverprofile=coverage.out

## Интеграционные тесты репозиториев поверх настоящего Postgres в Docker
## (testcontainers-go сам поднимет и остановит контейнер).
test-integration:
	go test -tags=integration ./internal/server/storage/postgres/... -v

## Показать процент покрытия тестами по всему проекту
cover: test
	go tool cover -func=coverage.out | tail -1

## Статический анализ стандартными средствами Go
vet:
	go vet ./...

## Форматирование кода
fmt:
	gofmt -l -w .

## default_api_level=API_HYBRID — генерировать сообщения в промежуточном
## (Hybrid) режиме миграции на Opaque API: поля остаются экспортируемыми
## (для совместимости с protoc-gen-grpc-gateway, который сам пишет в поля
## напрямую и не умеет в полный Opaque), но параллельно генерируются
## Get*/Set*/Has*/Clear* — весь наш код использует именно их, а не прямой
## доступ к полям. Это официально документированная Google промежуточная
## ступень миграции (API_OPEN -> API_HYBRID -> API_OPAQUE), а не костыль.
proto:
	protoc -I. -Ithird_party/googleapis \
	       --go_out=. --go_opt=paths=source_relative --go_opt=default_api_level=API_HYBRID \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       --grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
	       --openapiv2_out=. \
	       $(PROTO_FILES)