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
- **Backend**: Go with `net/http` standard library, SQLite via `modernc.org/sqlite` (pure Go, no CGO).
  Every build is `CGO_ENABLED=0`; keep it that way — the Android app runs the same static binary.
  The driver's DSN options (`driverOptions` in `internal/db/db.go`) make timestamps read and write
  exactly as the old `mattn/go-sqlite3` build did; `internal/db/testdata/mattn-go-sqlite3.db` pins it.
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
internal/display/    - Minion 3 e-paper frame renderer: pure, deterministic, golden-tested (docs/MINION3_DISPLAY.md)
internal/version/    - Application version (stamped Date + Patch, from the commit)
internal/serve/      - Flags/env → bind address, discovery, graceful SIGTERM shutdown
internal/discovery/  - UDP discovery responder (HOMEAPI-DISCOVER/1 on 9998), mirrored by the app
internal/hostenv/    - HOMEAPI_TZ → time.Local (Go on Android starts in UTC); inert when unset
apps/android/        - The Android app: thin client or the same binary on the phone (apps/android/README.md)
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
- `GET    /api/displays/minion3/frame.bin` - Minion 3 e-paper frame, 4000 bytes panel-native (query: at, wake)
- `GET    /api/displays/minion3/frame.png` - Same frame as PNG (query: at, wake, scale)

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

## Minion 3 Display
`internal/display` renders a 250×122 1-bit dashboard from the `minion-params`,
`minion-quotes` and `minion-sum` entries; `internal/api/display.go` serves it.
Rendering is a pure function of (config, inputs, now) and is pinned by
`internal/display/testdata/golden.pbm` — regenerate with
`go test ./internal/display -update` and eyeball the PBM when a layout change is
intended. The device contract (headers, bit packing, sleep computation) is in
`docs/MINION3_DISPLAY.md`; keep it additive. Only a `frame.bin` request without
`at=` writes anything (`minion3-prev-quotes`, `minion3-last-seen`); PNG previews
and `at=` renders are read-only.

## Versioning
`vYEAR.MONTH.PATCH` — a calendar version, where PATCH is the repository's commit
count, so every commit is a patch release. YEAR.MONTH is the month of HEAD's
**committer date, in the committer's own timezone** — there are no constants to
bump, and it is never the build clock (only an unstamped `go build`, which has
no commit to ask, falls back to the build month under patch 0). The month is
not zero-padded, which keeps the string valid semver. `scripts/version.mjs` is
the one place it is assembled: `--ldflags` prints the `-X …version.Patch/Date`
flags every Go build is stamped with (Makefile, the release workflow, the
Android app's server build), the web bundle reads it as `REACT_APP_VERSION`,
and the APK takes it as `-PappVersionName` / `-PappVersionCode`, so the GUI
header, `--version`, `/api/health` and the app always agree. A build without
git, or a shallow clone, reports patch `0`; anything building a release needs
the full commit graph (`fetch-depth: 0`). Don't assert a literal version in a
test — it changes with every commit.

Releases: `.github/workflows/release.yml` runs on `v*` tags, pushes to `main`
and manual dispatch. The gate is a `## vX.Y.Z` section in `CHANGELOG.md` that
matches the commit's version (a push without one is skipped, a tag or dispatch
without one fails); it publishes `homeapi-linux-amd64`, `homeapi-linux-arm64`
(static, frontend embedded), `homeapi.apk` and `SHA256SUMS`. Keep those asset
names — `releases/latest/download/homeapi.apk` is linked from the docs.

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
