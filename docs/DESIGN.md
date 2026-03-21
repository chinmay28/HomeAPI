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
- Complex data types (only text key-value pairs)
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

### 3.2 Entry Model

```go
type Entry struct {
    ID        int64     `json:"id"`
    Category  string    `json:"category"`
    Key       string    `json:"key"`
    Value     string    `json:"value"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
}
```

### 3.3 Constraints
- `category + key` must be unique (upsert semantics available)
- `category` defaults to "default" if not specified
- `key` is required and cannot be empty
- `value` can be empty string

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
- Returns 201 Created with the created entry
- Returns 409 Conflict if category+key already exists

#### Get Entry
```
GET /api/entries/42
```
- Returns 200 with the entry
- Returns 404 if not found

#### Update Entry
```
PUT /api/entries/42
Content-Type: application/json

{"value": "Apple Inc. - Updated"}
```
- Partial updates allowed (only specified fields are changed)
- Returns 200 with the updated entry
- `updated_at` is automatically set

#### Delete Entry
```
DELETE /api/entries/42
```
- Returns 204 No Content on success
- Returns 404 if not found

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
- Returns 200 with `{"status": "ok", "version": "1.0.0"}`

### 4.2 Error Responses

All errors follow a consistent format:
```json
{
    "error": "Human-readable error message",
    "code": "VALIDATION_ERROR"
}
```

Error codes: `NOT_FOUND`, `VALIDATION_ERROR`, `CONFLICT`, `INTERNAL_ERROR`

### 4.3 CORS
CORS is enabled for all origins in development. In production, the frontend is served from the same origin so CORS is not needed.

## 5. Frontend Design

### 5.1 Pages

1. **Dashboard** (`/`): Overview showing categories with entry counts
2. **Entries List** (`/entries`): Filterable, searchable table of entries
3. **Entry Detail** (`/entries/:id`): View/edit a single entry
4. **Import/Export** (`/settings`): Import and export functionality

### 5.2 Components

- `Header`: Navigation bar with links
- `EntryTable`: Sortable, filterable table of entries
- `EntryForm`: Create/edit entry form
- `CategorySidebar`: Category filter panel
- `SearchBar`: Global search input
- `ImportExport`: Import/export controls
- `Notification`: Toast notifications for success/error

### 5.3 State Management
React hooks (`useState`, `useEffect`) with a simple API client module. No Redux needed for this scope.

## 6. Build & Deployment

### 6.1 Build Process

```
1. cd frontend && npm run build     # Build React app
2. go build -o homeapi ./cmd/homeapi  # Build Go binary (embeds frontend)
```

The Makefile orchestrates this into a single `make build` command.

### 6.2 Configuration

| Environment Variable | Default | Description |
|---------------------|---------|-------------|
| `HOMEAPI_PORT` | `8080` | HTTP listen port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | Database file path |
| `HOMEAPI_LOG_LEVEL` | `info` | Log level (debug, info, warn, error) |

### 6.3 Deployment
1. Copy binary to target machine
2. Run: `./homeapi`
3. Access: `http://localhost:8080`

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
