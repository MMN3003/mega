APP_NAME=swap
CMD_DIR=./
BUILD_DIR=./build
GO_FILES=$(shell find . -name '*.go' -not -path "./vendor/*")
DOCKER_IMAGE=swap-backend:latest

# Default target
.PHONY: all
all: build

## Build binary
.PHONY: build
build:
	@echo ">> Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

## Run locally
.PHONY: run
run:
	@echo ">> Generating swagger docs..."
	swag init -g $(CMD_DIR)/main.go -o ./docs
	@echo ">> Running $(APP_NAME)..."
	go run $(CMD_DIR)

## Run with hot reload (requires air)
.PHONY: dev
dev:
	@echo ">> Running with hot reload..."
	air

## Run tests
.PHONY: test
test:
	@echo ">> Running tests..."
	go test ./... -cover -race

## Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo ">> Linting..."
	golangci-lint run ./...

## Docker build
docker-build:
	@echo ">> Generating swagger docs..."
	swag init -g $(CMD_DIR)/main.go -o ./docs
	@echo ">> Building Docker image..."
	docker build -t $(DOCKER_IMAGE) .


## Docker run
.PHONY: docker-run
docker-run:
	@echo ">> Running Docker container..."
	docker run --rm -it -p 8080:8080 --env-file .env $(DOCKER_IMAGE)

## Compose up
.PHONY: compose-up
compose-up:
	@echo ">> Starting services with docker-compose..."
	docker-compose up -d

## Compose down
.PHONY: compose-down
compose-down:
	@echo ">> Stopping services..."
	docker-compose down

## Swagger docs generation (requires swag CLI)
.PHONY: swagger
swagger:
	@echo ">> Generating swagger docs..."
	swag init -g $(CMD_DIR)/main.go -o ./docs

## Clean build artifacts
.PHONY: clean
clean:
	@echo ">> Cleaning..."
	rm -rf $(BUILD_DIR)
