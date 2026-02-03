# MWork API

> Go backend for MWork platform

## Quick Start

```bash
# Install dependencies
go mod download

# Run with hot reload
make dev

# Run tests
make test

# Apply migrations
make migrate-up
```

## Project Structure

```
mwork-api/
├── cmd/
│   └── api/main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration
│   ├── domain/              # Business logic modules
│   │   ├── auth/
│   │   ├── user/
│   │   ├── casting/
│   │   └── response/
│   ├── middleware/          # HTTP middlewares
│   └── pkg/                 # Shared utilities
├── migrations/              # Database migrations
├── docker/                  # Docker configs
└── api/                     # OpenAPI specs
```

## Tech Stack 

- **Go 1.22+**
- **Chi** - HTTP router
- **sqlx** - Database access
- **PostgreSQL 16** - Primary database 
- **Redis 7** - Cache & sessions 
