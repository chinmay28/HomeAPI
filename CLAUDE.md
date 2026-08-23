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
internal/version/    - Application version (Year/Month constants, stamped Patch)
frontend/            - React application
scripts/version.mjs  - Assembles the version for the Go and web builds
docs/                - Design document and user guide
tests/integration/   - Integration tests (API-level)
tests/e2e/           - End-to-end tests
```

## Data Model
Each entry has: `id`, `category`, `key`, `value`, `created_at`, `updated_at`.
Categories group related entries (e.g., "watchlist", "config", "notes").
The `value` is stored as plain text in the database regardless of content.

## API Endpoints
- `GET    /api/entries`               - List entries (query: ?category=X&search=X)
- `POST   /api/entries`               - Create entry
- `GET    /api/entries/:id_or_key`    - Get entry by numeric ID or key string
- `PUT    /api/entries/:id_or_key`    - Update entry by numeric ID or key string
- `DELETE /api/entries/:id_or_key`    - Delete entry by numeric ID or key string
- `DELETE /api/entries`               - Bulk delete (query/body: keys, ids, category, search, all, dry_run)
- `GET    /api/categories`            - List all categories
- `GET    /api/export`                - Export all data as JSON
- `POST   /api/import`                - Import data from JSON
- `GET    /api/health`                - Health check

## Value Field Behavior
The `value` field behaves differently on input vs output:

**API response** — `value` is always a JSON value:
- If the stored string is a valid JSON object or array, it is embedded as-is.
- Otherwise it is wrapped: `{"data": "stored string"}`.

**API response** — `value_text` is the stored string byte-for-byte (whitespace,
indentation, key order). `value` is parsed JSON, so any client that decodes it
has already lost the author's formatting; the GUI reads and edits from
`value_text` so opening and saving an entry never reflows it. Additive field.

**API input** (POST/PUT) — `value` accepts any JSON type:
- A JSON string `"San Jose"` is unwrapped and stored as the plain text `San Jose`.
- A JSON object or array `{"lat": 37.3}` is serialized and stored as the JSON string `{"lat": 37.3}`.

**Key uniqueness** — keys are treated as globally unique identifiers. You can use
`/api/entries/city` instead of `/api/entries/1`. Numeric path segments are
resolved as IDs first; non-numeric segments are resolved as keys.

## Versioning
`vYEAR.MONTH.PATCH` — a calendar version, where PATCH is the repository's commit
count, so every commit is a patch release. Year/month are Go source constants in
`internal/version/version.go` (bump by hand when a release line opens, never
from the build clock); the patch number is stamped at build time. The month is
not zero-padded, which keeps the string valid semver. `scripts/version.mjs` is the one place it is assembled, and both the
Go binary (`-ldflags -X`) and the web bundle (`REACT_APP_VERSION`) read from it,
so the GUI header, `--version`, and `/api/health` always agree. A build without
git reports patch `0`.

The application version is *not* the export/import format version — that is a
separate `version` field and stays at `"1"`.

## Code Style
- Go: standard `gofmt` formatting, error wrapping with `fmt.Errorf("context: %w", err)`
- React: functional components with hooks, no class components
- CSS: design tokens as custom properties in `index.css`; no inline `style` props
  for anything a class can express, and no colour literals outside the tokens.
  Dark mode is `:root[data-theme='dark']` — declare each token once, there.
- Per-device UI preferences (theme, featured stats) go in `localStorage`, never
  in the entries table: entries are the user's data and show up in
  `/api/entries`, the category list, and every export.
- Tests: table-driven tests in Go, descriptive test names

## Backward Compatibility — IMPORTANT
This project must remain backward compatible with existing deployments and stored data.

**Rules for all agents and contributors:**
1. **Never change the database schema in a breaking way.** Existing `.db` files must
   continue to work after any upgrade. New columns must have defaults; old columns
   must not be removed or renamed. All schema changes must go through additive
   migrations only.
2. **Never change the storage format of existing data.** Values are stored as plain
   text strings in SQLite. Do not re-encode, migrate, or transform existing rows.
3. **Never remove or rename API fields** that already exist in responses. Adding new
   fields is fine; removing or renaming breaks existing scripts and clients.
4. **Never change the meaning of existing API endpoints.** Numeric IDs still resolve
   entries by ID. Existing curl scripts using `/api/entries/1` must keep working.
5. **Never change the export/import JSON format** in a way that makes existing backup
   files unreadable. The `version` field exists precisely to allow future format
   changes — increment it and handle old versions if the format must change.
6. **Test backward compat explicitly** when touching the db layer, API response
   shapes, or import/export code.
