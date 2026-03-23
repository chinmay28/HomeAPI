# HomeAPI

A self-hosted REST API and web GUI for storing simple key-value text data. Compiles to a single static binary with an embedded React frontend.

**Use cases:** stock ticker watchlists, home automation configs, bookmarks, notes, and any simple text data that needs to be read/written by scripts or humans.

## Quick Start

```bash
# Install prerequisites (Go, Node.js, GCC) if needed
./scripts/install-prereqs.sh

# Build
make build

# Run
./homeapi

# Open http://localhost:8080
```

## Features

- **Single binary** - frontend embedded at compile time, zero runtime dependencies
- **REST API** - simple JSON API for scripts and automation (`curl`-friendly)
- **Web GUI** - dashboard, entry management, search, and filtering
- **Categories** - organize entries into groups (e.g. "watchlist", "config", "notes")
- **Search** - full-text search across keys and values
- **Import/Export** - backup and restore data as JSON
- **SQLite storage** - reliable, single-file database

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/entries?category=X&search=X&page=1&per_page=50` | List entries |
| `POST` | `/api/entries` | Create entry |
| `GET` | `/api/entries/:id` | Get entry |
| `PUT` | `/api/entries/:id` | Update entry |
| `DELETE` | `/api/entries/:id` | Delete entry |
| `GET` | `/api/categories` | List categories with counts |
| `GET` | `/api/export` | Export all data as JSON |
| `POST` | `/api/import` | Import data (`mode`: "merge" or "replace") |
| `GET` | `/api/health` | Health check |

## API Examples

```bash
# Create an entry
curl -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."}'

# List entries in a category
curl "http://localhost:8080/api/entries?category=watchlist"

# Search
curl "http://localhost:8080/api/entries?search=apple"

# Update
curl -X PUT http://localhost:8080/api/entries/1 \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'

# Delete
curl -X DELETE http://localhost:8080/api/entries/1

# Export for backup
curl http://localhost:8080/api/export -o backup.json

# Import
curl -X POST http://localhost:8080/api/import \
  -H "Content-Type: application/json" \
  -d @backup.json
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOMEAPI_PORT` | `8080` | HTTP listen port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | SQLite database file path |
| `HOMEAPI_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

The database directory and file are created automatically on first run.

## Build Requirements

- **Go** 1.21 or later
- **Node.js** 18+ and npm (for frontend build)
- **GCC** (required by the SQLite driver via CGO)

Run `./scripts/install-prereqs.sh` to install all prerequisites automatically. Supports Ubuntu/Debian, Fedora, RHEL/CentOS, Arch Linux, and macOS (Homebrew).

```bash
# Full build (frontend + backend)
make build

# Run tests (unit + integration + e2e)
make test

# Individual test levels
make test-unit
make test-integration
make test-e2e

# Development: run backend only (use `cd frontend && npm start` for frontend dev server)
make dev

# Clean build artifacts
make clean
```

## Project Structure

```
cmd/homeapi/           Main entrypoint and embed configuration
internal/api/          HTTP handlers and router
internal/db/           SQLite database access layer
internal/models/       Data models and validation
internal/middleware/    HTTP middleware (CORS, logging)
frontend/src/          React application source
docs/                  Design document and user guide
tests/integration/     Integration tests (API-level)
tests/e2e/             End-to-end tests (full workflow)
```

## Documentation

- [Design Document](docs/DESIGN.md) - architecture, data model, API design
- [User Guide](docs/USER_GUIDE.md) - detailed usage instructions and examples

## License

See [LICENSE](LICENSE).
