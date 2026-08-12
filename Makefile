.PHONY: build build-server build-client run-server run-client test vet fmt cover proto migrate-up migrate-down migrate-create

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
SERVER_ADDR  ?= localhost:50051

## Собрать server и client в bin/
build: build-server build-client

build-server:
	go build -o $(BIN_DIR)/gophkeeper-server ./cmd/server

build-client:
	go build -o $(BIN_DIR)/gophkeeper-client ./cmd/client

## Запустить сервер/клиент без сборки бинарника
run-server:
	DATABASE_DSN=$(DATABASE_DSN) JWT_SECRET=$(JWT_SECRET) GRPC_PORT=$(GRPC_PORT) go run ./cmd/server

run-client:
	GOPHKEEPER_SERVER=$(SERVER_ADDR) go run ./cmd/client

## Накатить все непримененные миграции на базу из DATABASE_DSN
migrate-up:
	migrate -database "$(DATABASE_DSN)" -path migrations up

## Откатить последнюю миграцию
migrate-down:
	migrate -database "$(DATABASE_DSN)" -path migrations down 1

## Создать пару файлов новой миграции: make migrate-create name=add_foo
migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

## Юнит-тесты со сбором покрытия в coverage.out
test:
	go test ./... -race -coverprofile=coverage.out

## Показать процент покрытия тестами по всему проекту
cover: test
	go tool cover -func=coverage.out | tail -1

## Статический анализ стандартными средствами Go
vet:
	go vet ./...

## Форматирование кода
fmt:
	gofmt -l -w .

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       $(PROTO_FILES)