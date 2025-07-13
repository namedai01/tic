.PHONY: run build test clean docker-up docker-down migrate-up migrate-down swagger

# Variables
BINARY_NAME=tic-knowledge-system
MAIN_PACKAGE=./cmd/server

# Development
run:
	go run $(MAIN_PACKAGE)/main.go

build:
	go build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

test:
	go test -v ./...

clean:
	rm -rf bin/

# Dependencies
deps:
	go mod download
	go mod tidy

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-build:
	docker build -t $(BINARY_NAME) .

# Database
migrate-up:
	@echo "Auto-migration handled by GORM"

migrate-down:
	@echo "Manual migration down required"

seed:
	go run ./cmd/seed/main.go

# Documentation
swagger:
	swag init -g cmd/server/main.go -o docs/

# Development setup
setup: deps docker-up
	@echo "Waiting for databases to be ready..."
	sleep 10
	@echo "Setup complete! Run 'make run' to start the server"

# Production
prod-build:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

# Lint
lint:
	golangci-lint run

# Format
fmt:
	go fmt ./...
