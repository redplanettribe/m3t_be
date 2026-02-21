include .env
export

run:
	go run cmd/api/main.go

watch:
	air

migrate-up:
	migrate -path migrations -database "$(DATABASE_URL)" up

migrate-down:
	migrate -path migrations -database "$(DATABASE_URL)" down

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

swag:
	swag init -g cmd/api/main.go -o docs

docker-up:
	docker compose up -d

docker-down:
	docker compose down

start-dev: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 3
	$(MAKE) migrate-up
	$(MAKE) watch
