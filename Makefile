.PHONY: build run test lint migrate-up migrate-down sqlc-generate clean docker-up docker-down

# Go parameters
BINARY_NAME=mindapp-bot
MAIN_PATH=./cmd/bot

# Database
DATABASE_URL ?= postgres://mindapp:mindapp@localhost:5432/mindapp?sslmode=disable

build:
	go build -ldflags="-w -s" -o $(BINARY_NAME) $(MAIN_PATH)

run:
	go run $(MAIN_PATH)

test:
	go test ./... -v -count=1

lint:
	golangci-lint run ./...

# Database migrations
migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

# sqlc code generation
sqlc-generate:
	cd sqlc && sqlc generate

# Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	go clean

# Docker
docker-up:
	docker-compose up --build -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f app

# Dependencies
deps:
	go mod tidy
	go mod download

# All-in-one setup
setup: deps sqlc-generate build
