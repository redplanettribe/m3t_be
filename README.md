# Multi-Track Ticketing Backend API

This repository contains the backend API for the Multi-Track Ticketing system.

## 🤖 AI Agent Context & Rules
**If you are an AI assistant working on this codebase, please strictly adhere to the following rules:**

### 1. Technology Stack
- **Language**: Golang (Latest version)
- **Database**: PostgreSQL
- **ORM**: **NO ORM**. Use `database/sql` with raw SQL queries.
- **Migrations**: `golang-migrate`
- **Documentation**: Swagger/OpenAPI (using `swaggo`)
- **Transport**: Standard `net/http` 

### 2. Architecture: Clean Architecture
We adhere to a strict **Clean Architecture** separation of concerns.
- **`internal/domain`**: Entities and Interface Definitions (Repository, UseCase). **No external dependencies** (except time/context).
- **`internal/usecase`**: Business logic. Depends only on `domain`.
- **`internal/repository`**: Data access implementation (e.g., `postgres`). Depends on `domain`.
- **`internal/delivery`**: Transport layer (e.g., `http`). Depends on `usecase`.

### 3. Workflow
- **Migrations**: Always create up/down migration files in `migrations/` for DB schema changes.
- **Tests**: (Future) Table-driven tests preferred.

---

## 📂 Project Structure

```text
├── cmd
│   └── api              # Application entry point (main.go)
├── config               # Configuration setup
├── docs                 # Generated Swagger docs
├── internal
│   ├── domain           # Core business entities & interface definitions
│   ├── usecase          # Application business logic
│   ├── repository       # Database implementations (Raw SQL)
│   └── delivery
│       └── http         # HTTP Handlers
├── migrations           # SQL migration files
├── Makefile             # Command runner
└── docker-compose.yml   # Local development infrastructure
```

## 🚀 Getting Started

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- Make

### Running Locally
We have a convenient Makefile to handle the developer workflow.

1. **Start Environment (DB + App)**
   This will start Postgres in Docker, run migrations, and start the API.
   ```bash
   make start-dev
   ```

2. **Useful Commands**
   - `make docker-up`: Start Postgres container.
   - `make migrate-up`: Run all pending migrations.
   - `make run`: Run the Go application.
   - `make swag`: Regenerate Swagger documentation.

### 📚 Documentation
Once running, Swagger UI is available at:
[http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)
