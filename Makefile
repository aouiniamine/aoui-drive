.PHONY: build run dev watch test clean docker-up docker-down tidy sqlc create-client swagger

APP_NAME := aoui-drive
BUILD_DIR := ./bin

build:
	@echo "Building $(APP_NAME)..."
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/$(APP_NAME)

run: build
	@./$(BUILD_DIR)/$(APP_NAME)

dev:
	@go run github.com/air-verse/air@latest

watch: dev

test:
	@go test -v ./...

test-coverage:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

clean:
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f

tidy:
	@go mod tidy

lint:
	@golangci-lint run

sqlc:
	@go run github.com/sqlc-dev/sqlc/cmd/sqlc@latest generate

create-client:
	@go run ./cmd/create-client -name="$(NAME)" -role="$(ROLE)"

swagger:
	@go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/aoui-drive/main.go -o docs

setup: docker-up sqlc tidy swagger
	@cp -n .env.example .env 2>/dev/null || true
	@mkdir -p data
	@echo "Setup complete! Run 'make dev' to start the server."
