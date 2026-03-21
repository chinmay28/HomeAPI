# HomeAPI - Development Guide

## Project Overview
HomeAPI is a self-hosted REST API and web GUI for storing simple key-value text data.
Use cases: stock ticker watchlists, home automation configs, bookmarks, notes, and any simple text data.

## Build & Run Commands
- **Build**: `make build` (produces single static binary with embedded frontend)
- **Dev backend**: `go run ./cmd/homeapi`
- **Dev frontend**: `cd frontend && npm start`
- **Run tests**: `make test`
- **Unit tests only**: `go test ./internal/...`
- **Integration tests**: `go test ./tests/integration/...`
- **E2E tests**: `go test ./tests/e2e/...`
- **Lint**: `golangci-lint run ./...`
- **Frontend lint**: `cd frontend && npm run lint`
- **Frontend tests**: `cd frontend && npm test`

## Architecture
- **Backend**: Go with `net/http` standard library, SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- **Frontend**: React (Create React App), embedded into Go binary via `embed.FS`
- **Database**: SQLite, stored at `~/.homeapi/homeapi.db` by default (configurable via `HOMEAPI_DB_PATH`)
- **Single binary**: Frontend is built and embedded at compile time

## Project Structure
```
cmd/homeapi/         - Main entrypoint
internal/api/        - HTTP handlers and router
internal/db/         - Database access layer
internal/models/     - Data models
internal/middleware/  - HTTP middleware (CORS, logging, auth)
frontend/            - React application
docs/                - Design document and user guide
tests/integration/   - Integration tests (API-level)
tests/e2e/           - End-to-end tests
```

## Data Model
Each entry has: `id`, `category`, `key`, `value`, `created_at`, `updated_at`.
Categories group related entries (e.g., "watchlist", "config", "notes").

## API Endpoints
- `GET    /api/entries`             - List entries (query: ?category=X&search=X)
- `POST   /api/entries`             - Create entry
- `GET    /api/entries/:id`         - Get entry by ID
- `PUT    /api/entries/:id`         - Update entry
- `DELETE /api/entries/:id`         - Delete entry
- `GET    /api/categories`          - List all categories
- `GET    /api/export`              - Export all data as JSON
- `POST   /api/import`             - Import data from JSON
- `GET    /api/health`              - Health check

## Code Style
- Go: standard `gofmt` formatting, error wrapping with `fmt.Errorf("context: %w", err)`
- React: functional components with hooks, no class components
- Tests: table-driven tests in Go, descriptive test names
