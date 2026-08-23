# HomeAPI - User Guide

## Table of Contents
1. [Getting Started](#getting-started)
2. [Using the Web Interface](#using-the-web-interface)
3. [Using the REST API](#using-the-rest-api)
4. [Working with JSON Values](#working-with-json-values)
5. [Import and Export](#import-and-export)
6. [Configuration](#configuration)
7. [Examples](#examples)

## Getting Started

### Installation

Download the latest binary for your platform from the releases page, or build from source:

```bash
# Build from source
make build

# The binary is at ./homeapi
```

### Running

```bash
# Start with defaults (port 9999, database in ~/.homeapi/)
./homeapi

# Custom port
HOMEAPI_PORT=3000 ./homeapi

# Custom database location
HOMEAPI_DB_PATH=/data/mydata.db ./homeapi
```

Open your browser to `http://localhost:9999` to access the web interface.

## Using the Web Interface

The interface works the same on a phone as on a desktop. On a narrow screen the
navigation moves to a bottom tab bar, tables reflow into stacked cards, and a
round **+** button in the bottom-right corner creates a new entry. It follows
your system's light or dark theme automatically; **Settings → Appearance** can
pin it to one or the other.

The version under the **HomeAPI** wordmark is the build you're running; the
dashboard's **Server** card shows the version the *server* reports, so a
mismatch means the page is stale — reload it.

### Dashboard
The dashboard opens with your total entry count, category count, and server
health, followed by all your categories with their entry counts. Click a
category to view its entries.

**Featured stats** — click **Customize stats** to choose which cards appear:

| Stat | Shows |
|---|---|
| Total entries | Every entry, across all categories |
| Categories | How many categories exist |
| Server status | Whether the API is reachable, and its version |
| Largest category | The category with the most entries |
| *category name* | The entry count for one category you care about |
| *entry key* | The **value** of one entry, e.g. `minion-sum` |

Entry cards are the point if what you want on the home screen is your data
rather than counts: tick an entry under **Entry values** and its current value
is the card. Long values are trimmed to fit; clicking the card opens the entry.
When you have more than a handful of entries, the search box above the list
finds them by key or value.

Cards appear in the order you tick them. The choice is saved in that browser, so
your phone and your laptop can feature different things. **Reset to default**
puts back the original three.

### Browsing Entries
- Use the **category filter** to narrow by category
- Use the **search bar** to find entries by key or value
- Click any entry's key to view its details

### Creating Entries
1. Click **"New entry"** in the header (or the **+** button on a phone)
2. Fill in:
   - **Category**: Group name (e.g., "watchlist", "config", "notes")
   - **Key**: Unique identifier within the category (e.g., "AAPL", "thermostat_temp")
   - **Value**: The data to store — plain text or a JSON object
3. Click **"Save"**

### Editing Entries
1. Click on an entry to open it
2. Modify the fields
3. Click **"Save"**

### Working with JSON in the GUI
The value box is a plain text area, and what you type is what gets stored —
line breaks and indentation included. Formatting a JSON value by hand is safe:
open the entry again and it comes back exactly as you left it, and saving an
entry you didn't touch changes nothing.

When the value parses as JSON, a **Format JSON** button appears above the box
and reindents it with two spaces. JSON that was written on a single line —
by a `curl` script, say — is pretty-printed for reading on the entry page, but
only reformatted in storage if you press that button and save.

### Deleting Entries
1. Click on an entry to open it
2. Click **"Delete"**
3. Confirm the deletion

## Using the REST API

The REST API is available at `/api/` and is designed for easy use with `curl` and scripts.

### List All Entries

```bash
curl http://localhost:9999/api/entries
```

### Filter by Category

```bash
curl "http://localhost:9999/api/entries?category=watchlist"
```

### Search Entries

```bash
curl "http://localhost:9999/api/entries?search=apple"
```

### Create an Entry

```bash
curl -X POST http://localhost:9999/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."}'
```

### Get an Entry

You can look up entries by numeric ID **or by key**:

```bash
# By numeric ID
curl http://localhost:9999/api/entries/1

# By key — much easier to remember and script
curl http://localhost:9999/api/entries/AAPL
curl http://localhost:9999/api/entries/thermostat_temp
```

### Update an Entry

```bash
# By numeric ID
curl -X PUT http://localhost:9999/api/entries/1 \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'

# By key
curl -X PUT http://localhost:9999/api/entries/AAPL \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'
```

### Delete an Entry

```bash
# By numeric ID
curl -X DELETE http://localhost:9999/api/entries/1

# By key
curl -X DELETE http://localhost:9999/api/entries/AAPL
```

### Bulk Delete Entries

`DELETE /api/entries` removes every entry matching the selectors you pass.
Selectors can be query parameters, a JSON body, or both.

```bash
# By key (repeat the parameter or comma-separate)
curl -X DELETE "http://localhost:9999/api/entries?key=AAPL&key=GOOGL"
curl -X DELETE "http://localhost:9999/api/entries?keys=AAPL,GOOGL"

# By numeric ID
curl -X DELETE "http://localhost:9999/api/entries?ids=1,2,3"

# By query: an entire category, or everything matching a search
curl -X DELETE "http://localhost:9999/api/entries?category=watchlist"
curl -X DELETE "http://localhost:9999/api/entries?search=apple"

# Combine selectors — category and search narrow the match
curl -X DELETE "http://localhost:9999/api/entries?category=watchlist&search=apple"

# Same thing with a JSON body
curl -X DELETE http://localhost:9999/api/entries \
  -H "Content-Type: application/json" \
  -d '{"keys": ["AAPL", "GOOGL"], "category": "watchlist"}'
```

**Selectors**

| Parameter | Meaning |
|-----------|---------|
| `key` / `keys` | Delete entries with these keys (repeatable or comma-separated) |
| `id` / `ids` | Delete entries with these numeric IDs |
| `category` | Restrict to (or select) a category |
| `search` | Restrict to entries whose key or value matches |
| `all=true` | Delete every entry — required when no other selector is given |
| `dry_run=true` | Report what would be deleted without deleting anything |

Keys and IDs are OR'd together; `category` and `search` further narrow the
match. A request with no selector at all is rejected with `400` so you cannot
empty the database by accident — pass `all=true` if that is what you want.

**Response**

```json
{
  "deleted": 2,
  "matched": 2,
  "dry_run": false,
  "entries": [
    {"id": 1, "category": "watchlist", "key": "AAPL"},
    {"id": 2, "category": "watchlist", "key": "GOOGL"}
  ]
}
```

Preview before deleting with `dry_run=true`, which lists the matches and
reports `"deleted": 0`:

```bash
curl -X DELETE "http://localhost:9999/api/entries?category=watchlist&dry_run=true"
```

Deleting nothing is not an error: an unmatched selector returns `200` with
`"deleted": 0`.

### Get All Categories

```bash
curl http://localhost:9999/api/categories
```

### Health Check

```bash
curl http://localhost:9999/api/health
```

```json
{"status": "ok", "version": "v2026.8.311"}
```

`version` is the running build, `vYEAR.MONTH.PATCH` — the patch number is the
repository's commit count. `./homeapi --version` prints the same string.

## Working with JSON Values

The `value` field supports both plain text and structured JSON data.

### Plain Text Values

When you store a plain string, the API wraps it in a `{"data": "..."}` envelope
so that the `value` field is always valid JSON:

```bash
$ curl -X POST http://localhost:9999/api/entries \
    -d '{"key": "city", "value": "San Jose"}'

# Response:
{
  "id": 1,
  "key": "city",
  "value": {"data": "San Jose"},
  ...
}
```

To read it back in a script:
```bash
curl -s http://localhost:9999/api/entries/city | jq '.value.data'
# → "San Jose"
```

### JSON Object / Array Values

When you store a JSON object or array, it is embedded directly in the response
without any wrapping:

```bash
$ curl -X POST http://localhost:9999/api/entries \
    -d '{"key": "location", "value": {"lat": 37.3, "lon": -121.9}}'

# Response:
{
  "key": "location",
  "value": {"lat": 37.3, "lon": -121.9},
  ...
}
```

Reading structured data:
```bash
LAT=$(curl -s http://localhost:9999/api/entries/location | jq '.value.lat')
```

### Updating to a JSON Value

```bash
curl -X PUT http://localhost:9999/api/entries/location \
  -H "Content-Type: application/json" \
  -d '{"value": {"lat": 37.77, "lon": -122.41}}'
```

### The Exact Stored Text (`value_text`)

`value` is parsed JSON, so anything that decodes it loses the indentation and
key order that were stored. Every response therefore also carries `value_text`:
the stored string exactly as it sits in the database.

```bash
$ curl -s http://localhost:9999/api/entries/location | jq -r '.value_text'
{
    "city": "San Jose",
    "lat": 37.33
}
```

Use `value` for reading data (`jq '.value.lat'`), and `value_text` when the text
itself matters — diffing against a file, or editing without reflowing it. The
GUI edits from `value_text`, which is why it never reformats an entry you did
not change.

## Import and Export

### Export via GUI
1. Go to the **Settings** page
2. Click **"Export Data"**
3. A JSON file will download automatically

### Export via API

```bash
# Save to file
curl http://localhost:9999/api/export -o homeapi-backup.json

# Pretty print
curl http://localhost:9999/api/export | jq .
```

### Import via GUI
1. Go to the **Settings** page
2. Click **"Choose File"** and select a JSON export file
3. Choose import mode:
   - **Merge**: Keep existing entries, only add new ones
   - **Replace**: Overwrite existing entries with matching category+key
4. Click **"Import"**

### Import via API

```bash
# Merge mode (default) - skip existing entries
curl -X POST http://localhost:9999/api/import \
  -H "Content-Type: application/json" \
  -d @homeapi-backup.json

# Replace mode - overwrite existing entries
curl -X POST http://localhost:9999/api/import \
  -H "Content-Type: application/json" \
  -d '{"entries": [...], "mode": "replace"}'
```

### Export Format

The export file is a JSON object with this structure:

```json
{
  "version": "1",
  "exported_at": "2024-01-15T10:30:00Z",
  "entries": [
    {
      "id": 1,
      "category": "watchlist",
      "key": "AAPL",
      "value": "Apple Inc.",
      "created_at": "2024-01-10T08:00:00Z",
      "updated_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

Note: the `value` field in the export file is the **raw stored string**, not the
JSON-wrapped form used by the regular API endpoints.

## Configuration

HomeAPI is configured via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `HOMEAPI_PORT` | `9999` | HTTP server port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | Path to SQLite database file |
| `HOMEAPI_LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |

The database file and its directory are created automatically on first run.

## Examples

### Stock Watchlist Script

```bash
#!/bin/bash
# Add stocks to watchlist
for ticker in AAPL GOOGL MSFT AMZN; do
  curl -s -X POST http://localhost:9999/api/entries \
    -H "Content-Type: application/json" \
    -d "{\"category\": \"watchlist\", \"key\": \"$ticker\", \"value\": \"active\"}"
done

# List watchlist keys
curl -s "http://localhost:9999/api/entries?category=watchlist" | jq '.entries[].key'

# Look up a specific stock by key
curl -s http://localhost:9999/api/entries/AAPL | jq '.value.data'
```

### Home Automation Config

```bash
# Set thermostat temperature
curl -s -X POST http://localhost:9999/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "config", "key": "thermostat_temp", "value": "72"}'

# Read it back by key — no ID lookup needed
TEMP=$(curl -s http://localhost:9999/api/entries/thermostat_temp | jq -r '.value.data')
echo "Setting thermostat to $TEMP"
```

### Storing Structured Data

```bash
# Store a JSON config object
curl -s -X POST http://localhost:9999/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "config", "key": "mqtt", "value": {"host": "192.168.1.10", "port": 1883}}'

# Read it back
curl -s http://localhost:9999/api/entries/mqtt | jq '.value.host'
# → "192.168.1.10"
```

### Backup Cron Job

```bash
# Add to crontab: daily backup at midnight
0 0 * * * curl -s http://localhost:9999/api/export > /backups/homeapi-$(date +\%Y\%m\%d).json
```
