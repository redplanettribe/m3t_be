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
