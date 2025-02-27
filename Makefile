.PHONY: all build run clean proto test client

# Переменные
BINARY_NAME=grpc-image-server
CLIENT_NAME=grpc-image-client
PROTO_DIR=pkg/api
SERVER_DIR=cmd/server
CLIENT_DIR=cmd/client

# Определение операционной системы
ifeq ($(OS),Windows_NT)
	BINARY_NAME := $(BINARY_NAME).exe
	CLIENT_NAME := $(CLIENT_NAME).exe
	RM_CMD := del /f
	MKDIR_CMD := if not exist storage mkdir storage
	RUN_PREFIX := .\
else
	RM_CMD := rm -f
	MKDIR_CMD := mkdir -p storage
	RUN_PREFIX := ./
endif

all: proto build

# Генерация кода из proto-файлов
proto:
	@echo "Generating code from proto-files..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/file.proto

# Сборка сервера
build-server:
	@echo "Server assembly..."
	go build -o $(BINARY_NAME) $(SERVER_DIR)/main.go

# Сборка клиента
build-client:
	@echo "Client assembly..."
	go build -o $(CLIENT_NAME) $(CLIENT_DIR)/client.go

# Сборка всего
build: build-server build-client

# Запуск сервера
run-server:
	@echo "Server start..."
	./$(BINARY_NAME)

# Запуск клиента
run-client:
	@echo "Client start..."
	./$(CLIENT_NAME)

# Очистка
clean:
	@echo "Clean..."
	go clean
	$(RM_CMD) $(BINARY_NAME)
	$(RM_CMD) $(CLIENT_NAME)

# Установка зависимостей
deps:
	@echo "Dependencies installation..."
	go mod tidy
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Создание директории для хранения файлов
create-storage:
	@echo "Create storage directory..."
	$(MKDIR_CMD)

# Запуск всего процесса
start: proto build create-storage run-server

# Кросс-компиляция для Windows из Linux
build-windows:
	@echo "Build for Windows..."
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME).exe $(SERVER_DIR)/main.go
	GOOS=windows GOARCH=amd64 go build -o $(CLIENT_NAME).exe $(CLIENT_DIR)/client.go

# Кросс-компиляция для Linux из Windows
build-linux:
	@echo "Build for Linux..."
	set GOOS=linux&& set GOARCH=amd64&& go build -o $(BINARY_NAME) $(SERVER_DIR)/main.go
	set GOOS=linux&& set GOARCH=amd64&& go build -o $(CLIENT_NAME) $(CLIENT_DIR)/client.go

# Справка
help:
	@echo "Available commands:"
	@echo "  make proto         - Generate code from proto-files"
	@echo "  make build         - Build server and client"
	@echo "  make build-server  - Build only server"
	@echo "  make build-client  - Build only client"
	@echo "  make run-server    - Start server"
	@echo "  make run-client    - Start client"
	@echo "  make test          - Start tests"
	@echo "  make clean         - Clean"
	@echo "  make deps          - Install dependencies"
	@echo "  make start         - Start all process"
	@echo "  make build-windows - Cross-compilation for Windows (from Linux)"
	@echo "  make build-linux   - Cross-compilation for Linux (from Windows)"
	@echo "  make help          - Show this help" 