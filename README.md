# HomeAPI

A self-hosted REST API and web GUI for storing simple key-value text data. Compiles to a single static binary with an embedded React frontend.

**Use cases:** stock ticker watchlists, home automation configs, bookmarks, notes, and any simple text data that needs to be read/written by scripts or humans.

## Quick Start (one line)

Deploy HomeAPI as a `systemd` service with a single command. The **same**
command installs a fresh copy or upgrades an existing one in place:

```bash
curl -fsSL https://raw.githubusercontent.com/chinmay28/homeapi/main/scripts/quickstart.sh | sudo bash
```

This installs build prerequisites, builds the binary, creates a dedicated
`homeapi` system user, and starts the service. Open http://localhost:9999.

**Upgrades are non-disruptive and lossless.** Re-running the command on an
existing install:

- keeps your data untouched — the SQLite database lives in a persistent data
  directory (`/var/lib/homeapi`) that is never overwritten by upgrades,
- takes a consistent database backup (to `/var/lib/homeapi/backups`) before
  swapping anything,
- swaps the binary atomically and **automatically rolls back** to the previous
  version if the new one fails its health check.

```bash
# Common operations after install
systemctl status homeapi          # service status
journalctl -u homeapi -f          # live logs
sudo systemctl restart homeapi    # restart
```

Override defaults with environment variables, e.g. a custom port:

```bash
curl -fsSL https://raw.githubusercontent.com/chinmay28/homeapi/main/scripts/quickstart.sh | sudo HOMEAPI_PORT=9090 bash
```

| Variable | Default | Description |
|----------|---------|-------------|
| `HOMEAPI_REF` | `main` | Git branch/tag/commit to deploy |
| `HOMEAPI_PORT` | `9999` | HTTP listen port |
| `HOMEAPI_USER` | `homeapi` | System user the service runs as |
| `HOMEAPI_PREFIX` | `/opt/homeapi` | Install dir for source + binary |
| `HOMEAPI_DATA_DIR` | `/var/lib/homeapi` | Persistent data dir (DB + backups) |

A reference unit file is available at [`deploy/homeapi.service`](deploy/homeapi.service).

## Build From Source

```bash
# Install prerequisites (Go, Node.js, GCC) if needed
./scripts/install-prereqs.sh

# Build
make build

# Run
./homeapi

# Open http://localhost:9999
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
curl -X POST http://localhost:9999/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."}'

# List entries in a category
curl "http://localhost:9999/api/entries?category=watchlist"

# Search
curl "http://localhost:9999/api/entries?search=apple"

# Update
curl -X PUT http://localhost:9999/api/entries/1 \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'

# Delete
curl -X DELETE http://localhost:9999/api/entries/1

# Export for backup
curl http://localhost:9999/api/export -o backup.json

# Import
curl -X POST http://localhost:9999/api/import \
  -H "Content-Type: application/json" \
  -d @backup.json
```

## Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOMEAPI_PORT` | `9999` | HTTP listen port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | SQLite database file path |
| `HOMEAPI_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

The database directory and file are created automatically on first run.

## Build Requirements

- **Go** 1.26 or later
- **Node.js** 24+ (LTS) and npm (for frontend build)
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
