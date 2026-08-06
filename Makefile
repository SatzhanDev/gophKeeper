.PHONY: build build-server build-client run-server run-client test vet fmt cover proto

BIN_DIR := bin
PROTO_FILES := $(wildcard api/proto/*/v1/*.proto)


## Собрать server и client в bin/
build: build-server build-client

build-server:
	go build -o $(BIN_DIR)/gophkeeper-server ./cmd/server

build-client:
	go build -o $(BIN_DIR)/gophkeeper-client ./cmd/client

## Запустить сервер/клиент без сборки бинарника
run-server:
	go run ./cmd/server

run-client:
	go run ./cmd/client

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