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

# Production migrations: use .env.prod (DATABASE_URL only, never commit secrets).
# Default sslmode=disable when not set (many hosted DBs e.g. Sevalla don't enable SSL).
migrate-up-prod:
	@test -f .env.prod || (echo "Error: .env.prod not found. Create it with DATABASE_URL for production." && exit 1)
	set -a && . ./.env.prod && set +a && \
	DB_URL="$$DATABASE_URL"; \
	echo "$$DB_URL" | grep -q 'sslmode=' || { \
	  echo "$$DB_URL" | grep -q '?' && DB_URL="$$DB_URL&sslmode=disable" || DB_URL="$$DB_URL?sslmode=disable"; \
	} && \
	migrate -path migrations -database "$$DB_URL" up

migrate-down-prod:
	@test -f .env.prod || (echo "Error: .env.prod not found. Create it with DATABASE_URL for production." && exit 1)
	set -a && . ./.env.prod && set +a && \
	DB_URL="$$DATABASE_URL"; \
	echo "$$DB_URL" | grep -q 'sslmode=' || { \
	  echo "$$DB_URL" | grep -q '?' && DB_URL="$$DB_URL&sslmode=disable" || DB_URL="$$DB_URL?sslmode=disable"; \
	} && \
 	migrate -path migrations -database "$$DB_URL" down

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

# C4 diagrams: Structurizr DSL viewer (http://localhost:8081)
c4-lite:
	docker run -it --rm -p 8081:8080 -v "$(CURDIR)/docs/c4:/usr/local/structurizr" structurizr/lite

start-dev: docker-up
	@echo "Waiting for database to be ready..."
	@sleep 3
	$(MAKE) migrate-up
	$(MAKE) watch
