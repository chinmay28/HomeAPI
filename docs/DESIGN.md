# HomeAPI - Detailed Design Document

## 1. Overview

HomeAPI is a lightweight, self-hosted application for storing and retrieving simple text-based key-value data. It serves two primary audiences:

1. **Automated scripts** that store/retrieve data via REST API calls
2. **Humans** who interact through a web-based GUI

### 1.1 Goals
- Single static binary deployment (zero external dependencies at runtime)
- Simple REST API suitable for curl/scripts
- Clean web GUI for human users
- Import/export for backup and migration
- Categorized storage for organizing different types of data
- Minimal resource usage suitable for running on a Raspberry Pi or NAS

### 1.2 Non-Goals
- Multi-user authentication (single-user system)
- Real-time collaboration
- Distributed storage

## 2. Architecture

```
┌─────────────────────────────────────────────┐
│              Single Go Binary               │
│                                             │
│  ┌──────────────┐    ┌───────────────────┐  │
│  │  Embedded     │    │   REST API        │  │
│  │  React SPA    │◄──►│   Handlers        │  │
│  │  (embed.FS)   │    │   (net/http)      │  │
│  └──────────────┘    └───────┬───────────┘  │
│                              │              │
│                       ┌──────▼───────────┐  │
│                       │  Database Layer   │  │
│                       │  (SQLite)         │  │
│                       └──────┬───────────┘  │
│                              │              │
│                       ┌──────▼───────────┐  │
│                       │  ~/.homeapi/     │  │
│                       │  homeapi.db      │  │
│                       └──────────────────┘  │
└─────────────────────────────────────────────┘
```

### 2.1 Technology Choices

| Component | Choice | Rationale |
|-----------|--------|-----------|
| Language | Go | Static compilation, excellent HTTP stdlib, embed support |
| Database | SQLite (modernc.org/sqlite) | Pure Go driver, no CGO needed, single file DB |
| HTTP | net/http (stdlib) | No external dependency, sufficient for this use case |
| Router | Custom mux | Simple pattern matching, avoids dependency |
| Frontend | React | Well-known, component-based, good tooling |
| Embedding | go:embed | Built-in Go feature for static assets |

### 2.2 Why No CGO
Using `modernc.org/sqlite` (a pure Go translation of SQLite) means the binary can be cross-compiled for any platform without a C compiler. This makes deployment trivial.

## 3. Data Model

### 3.1 Database Schema

```sql
CREATE TABLE IF NOT EXISTS entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    category   TEXT NOT NULL DEFAULT 'default',
    key        TEXT NOT NULL,
    value      TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(category, key)
);

CREATE INDEX idx_entries_category ON entries(category);
CREATE INDEX idx_entries_key ON entries(key);
```

Values are always stored as plain text strings in SQLite, regardless of whether
they represent JSON or plain strings. The API layer handles presentation.

### 3.2 Entry Model (internal)

```go
type Entry struct {
    ID        int64     `json:"id"`
    Category  string    `json:"category"`
    Key       string    `json:"key"`
    Value     string    `json:"value"`   // plain text as stored in DB
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.3 Entry Response (API)

The API response uses `entryResponse` (in `internal/api/handlers.go`), which
renders the `value` field as a JSON value rather than a raw string:

```json
{
  "id": 1,
  "category": "default",
  "key": "city",
  "value": {"data": "San Jose"},
  "created_at": "...",
  "updated_at": "..."
}
```

If the stored value is a valid JSON object or array it is embedded as-is:
```json
{
  "key": "location",
  "value": {"lat": 37.3, "lon": -121.9}
}
```

### 3.4 Constraints
- `category + key` must be unique (upsert semantics available via import)
- `category` defaults to "default" if not specified
- `key` is required and cannot be empty
- `value` can be empty string
- Keys are treated as globally unique identifiers for lookup purposes

## 4. API Design

### 4.1 RESTful Endpoints

All API endpoints are prefixed with `/api/`.

#### List Entries
```
GET /api/entries?category=watchlist&search=AAPL&page=1&per_page=50
```
- Query parameters are all optional
- `category`: filter by category
- `search`: search in key and value fields
- `page` / `per_page`: pagination (defaults: page=1, per_page=50)
- Response includes pagination metadata

#### Create Entry
```
POST /api/entries
Content-Type: application/json

{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."}
```
- `value` accepts any JSON type: a JSON string is unwrapped for storage;
  a JSON object or array is stored as its JSON string representation.
- Returns 201 Created with the created entry
- Returns 409 Conflict if category+key already exists

#### Get Entry
```
GET /api/entries/42          # by numeric ID
GET /api/entries/city        # by key (non-numeric path segment)
```
- Returns 200 with the entry
- Returns 404 if not found
- Numeric path segments are resolved as IDs; all others as keys

#### Update Entry
```
PUT /api/entries/42
PUT /api/entries/city
Content-Type: application/json

{"value": "San Francisco"}
```
- Partial updates allowed (only specified fields are changed)
- `value` accepts any JSON type (same rules as Create)
- Returns 200 with the updated entry
- `updated_at` is automatically set

#### Delete Entry
```
DELETE /api/entries/42
DELETE /api/entries/city
```
- Returns 204 No Content on success
- Returns 404 if not found

#### Bulk Delete Entries
```
DELETE /api/entries?keys=AAPL,GOOGL
DELETE /api/entries?category=watchlist&search=apple
DELETE /api/entries?all=true
```
- Selectors may be given as query parameters, as a JSON body, or both.
  Body fields override the equivalent query parameter; `keys`/`ids` are additive.
- `key`/`keys` and `id`/`ids` accept repeated parameters and comma-separated
  lists; they are OR'd together. `category` and `search` narrow the match.
- At least one selector is required. Without one the request returns 400
  unless `all=true` is passed explicitly, so an unfiltered call cannot empty
  the database by accident.
- `dry_run=true` reports the matches without deleting anything.
- Returns 200 with `{"deleted": N, "matched": N, "dry_run": false, "entries": [{"id","category","key"}]}`.
  On a dry run, `matched` is populated and `deleted` is 0.
- Matching nothing is not an error: returns 200 with `deleted: 0`.
- The select and the delete run in one transaction against the same IDs, so the
  reported entries are exactly the rows removed.
- Per-entry routes are unaffected: `DELETE /api/entries/42` and
  `DELETE /api/entries/city` keep their existing behavior.

#### List Categories
```
GET /api/categories
```
- Returns list of category names with entry counts
- Response: `[{"name": "watchlist", "count": 15}, ...]`

#### Export Data
```
GET /api/export
```
- Returns all entries as a JSON array
- Content-Disposition header set for file download
- Filename: `homeapi-export-YYYY-MM-DD.json`
- Values in the export are the raw stored strings (not JSON-wrapped)

#### Import Data
```
POST /api/import
Content-Type: application/json

{"entries": [...], "mode": "merge"}
```
- `mode`: "merge" (skip existing) or "replace" (overwrite existing)
- Returns summary: `{"imported": 42, "skipped": 3, "errors": 0}`

#### Health Check
```
GET /api/health
```
- Returns 200 with `{"status": "ok", "version": "v1.0.311"}`
- `version` is the application version, `vMAJOR.MINOR.PATCH` where the patch
  number is the repository's commit count (see §6.3). It is unrelated to the
  export/import format `version`, which is its own field and stays at `"1"`.

### 4.2 Value Field Encoding

The `value` field is always a JSON value in API responses:

| Stored text | API response `value` |
|---|---|
| `San Jose` | `{"data": "San Jose"}` |
| `72` | `{"data": "72"}` |
| `{"lat": 37.3, "lon": -121.9}` | `{"lat": 37.3, "lon": -121.9}` |
| `["a", "b"]` | `["a", "b"]` |

On input, the rules are symmetric:

| Request `value` | Stored text |
|---|---|
| `"San Jose"` (JSON string) | `San Jose` |
| `{"lat": 37.3}` (JSON object) | `{"lat": 37.3}` |
| `["a","b"]` (JSON array) | `["a","b"]` |

Responses also carry `value_text`: the stored string byte-for-byte, whitespace
and key order included. `value` cannot serve that purpose — it is parsed JSON,
and any client that decodes it (every browser does) has already lost the
formatting the author typed. The GUI reads and edits from `value_text`, which is
what keeps a hand-indented config from being reflowed by someone opening it and
pressing Save. Additive field; `value` keeps its meaning and its shape.

### 4.3 Error Responses

All errors follow a consistent format:
```json
{
    "error": "Human-readable error message",
    "code": "VALIDATION_ERROR"
}
```

Error codes: `NOT_FOUND`, `VALIDATION_ERROR`, `CONFLICT`, `INTERNAL_ERROR`

### 4.4 CORS
CORS is enabled for all origins in development. In production, the frontend is served from the same origin so CORS is not needed.

## 5. Frontend Design

### 5.1 Pages

1. **Dashboard** (`/`): user-selected featured stats over the category list
2. **Entries List** (`/entries`): Filterable, searchable table of entries
3. **Entry Detail** (`/entries/:id`): View/edit a single entry
4. **Settings** (`/settings`): appearance, import and export

The dashboard's featured stats are ids in `localStorage`: the four data-free
ones (`total`, `categories`, `server`, `largest`) plus `category:<name>` for a
per-category count. Ids that can't be rendered — a category since emptied, an
id from a newer build — are skipped rather than pruned, so a category that
comes back brings its card back with it.

### 5.2 App shell

`App.js` owns the chrome around the routed pages:

- **Header** (sticky): the brand lockup — app icon, wordmark, and the running
  build number underneath it — on the left; navigation, the primary "New entry"
  action, and the developer mark on the right.
- **Tab bar** (phones only): the same destinations as a bottom bar, with a
  floating action button for "New entry". Both are dismissed while the
  on-screen keyboard is up (`useKeyboardOpen`) so they never float over it.
- **Developer badge**: tapping the header mark throws the badge up full screen
  for three seconds. It is rendered outside the header because the header's
  `backdrop-filter` makes it a containing block, which would trap a fixed
  overlay inside the header strip.

Navigation collapses at 720px: the header nav hides, the tab bar appears, and
the shell becomes a fixed-height app frame with the main pane scrolling inside
it.

### 5.3 Look and feel

Design tokens (colours, radius, shadows) are CSS custom properties declared
once in `index.css`. The palette, radius, and header/tab-bar structure are
shared with [CountRoster](https://github.com/chinmay28/CountRoster) so the two
self-hosted tools read as one family.

**Theming.** `<html data-theme>` is always `light` or `dark`, resolved from the
saved preference (`homeapi.theme`, default `system`) or, for `system`, from
`prefers-color-scheme`. Because the attribute is always present, the dark
palette is declared once rather than duplicated across a media query and an
override. An inline script in `index.html` stamps it before the first paint so
a dark machine never flashes white while the bundle loads; `src/theme.js` owns
it afterwards and follows OS changes live while the preference is `system`. The
preference is a module-level store, not component state — the control lives on
the Settings page but every surface has to change with it, and a stale second
copy would clobber the choice on the next OS flip.

**Per-device preferences** (theme, featured dashboard stats) live in
`localStorage`, never in the entries table: they are view settings for one
browser, and storing them as entries would leak them into `/api/entries`, the
category list, and every export.

Mobile specifics: 44px minimum tap targets, 16px inputs (below that, iOS Safari
zooms on focus), `env(safe-area-inset-*)` padding for notched devices, and
tables that reflow into stacked, labelled cards rather than scrolling
sideways.

### 5.4 State Management
React hooks (`useState`, `useEffect`) with a simple API client module. No Redux needed for this scope.

The create-entry form's open state lives in the URL (`/entries?new=1`) so the
floating action button can open it from any page.

### 5.5 Static serving

Requests that name a real file in the embedded build get that file; everything
else gets `index.html` so client-side routes survive deep links and refreshes.
The shell is written out directly rather than by rewriting the path and
delegating to `http.FileServer` — that redirects any request ending in
`index.html` to `./`, which the browser resolves against the URL it asked for,
turning `/entries/9` into a redirect loop. The shell is served `no-cache`; the
bundles it points at carry content hashes and can be cached freely.

## 6. Build & Deployment

### 6.1 Build Process

```
1. cd frontend && npm run build     # Build React app
2. go build -o homeapi ./cmd/homeapi  # Build Go binary (embeds frontend)
```

The Makefile orchestrates this into a single `make build` command.

### 6.3 Versioning

The scheme is `vMAJOR.MINOR.PATCH`, where the patch number is the repository's
commit count — every commit is a patch release, so `v1.0.311` is the 311th
commit on the 1.0 line.

- `MAJOR`/`MINOR` are Go source constants in `internal/version/version.go`,
  bumped by hand. That file is the single declaration of them in the tree.
- `PATCH` only exists at build time (`git rev-list --count HEAD`). The Go
  binary gets it stamped in via `-ldflags -X`; the web bundle gets it inlined
  by Create React App from `REACT_APP_VERSION`.

Both sides call `scripts/version.mjs`, so the header, `homeapi --version`, and
`/api/health` can never disagree. A build made without git — a tarball, or a
shallow clone — reports patch `0`, which is deliberately a visible
non-release rather than a plausible-looking lie. Anything building a release
needs the full commit graph (`fetch-depth: 0`, or `--filter=blob:none` rather
than `--depth 1`).

### 6.2 Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOMEAPI_PORT` | `9999` | HTTP listen port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | Database file path |
| `HOMEAPI_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### 6.3 Deployment
1. Copy binary to target machine
2. Run: `./homeapi`
3. Access: `http://localhost:9999`

Database is created automatically on first run.

## 7. Testing Strategy

### 7.1 Unit Tests
- **Database layer**: Test CRUD operations using in-memory SQLite
- **API handlers**: Test with `httptest.NewRecorder()` and mock DB
- **Models**: Test validation logic
- Location: `*_test.go` files alongside source

### 7.2 Integration Tests
- **API integration**: Start real HTTP server with in-memory DB, test full request/response cycle
- **Import/Export**: Test round-trip of export then import
- Location: `tests/integration/`

### 7.3 End-to-End Tests
- **Full workflow**: Start server, create entries via API, verify via API, test export/import
- **Category management**: Create entries in multiple categories, verify filtering
- Location: `tests/e2e/`

## 8. Security Considerations

- No authentication by default (designed for local/trusted network use)
- SQL injection prevented by parameterized queries
- Input validation on all API inputs
- CORS restricted in production mode
- No sensitive data stored (text key-value pairs only)

## 9. Backward Compatibility

Backward compatibility with existing data and clients is a hard requirement.

### 9.1 Database
- Schema changes must be **additive only** (new columns with defaults, new indexes).
- Existing rows must never be transformed or migrated automatically.
- The `migrate()` function uses `CREATE TABLE IF NOT EXISTS` / `CREATE INDEX IF NOT EXISTS`
  and must remain safe to run against any existing database version.

### 9.2 API
- Existing response fields must not be removed or renamed.
- Numeric ID lookups (`/api/entries/1`) must continue to work forever.
- The export/import JSON format (field names, nesting) must remain stable.
  If a breaking format change is ever needed, increment `version` and support
  reading old versions.

### 9.3 Value encoding
- The `{"data": "..."}` wrapper for plain-text values is part of the public API
  contract. Scripts that parse `response.value.data` must not break.
- JSON object/array values embedded directly are also part of the contract.
