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
# Start with defaults (port 8080, database in ~/.homeapi/)
./homeapi

# Custom port
HOMEAPI_PORT=3000 ./homeapi

# Custom database location
HOMEAPI_DB_PATH=/data/mydata.db ./homeapi
```

Open your browser to `http://localhost:8080` to access the web interface.

## Using the Web Interface

### Dashboard
The dashboard shows all your categories with entry counts. Click a category to view its entries.

### Browsing Entries
- Use the **category filter** on the left to narrow by category
- Use the **search bar** to find entries by key or value
- Click any entry to view its details

### Creating Entries
1. Click the **"New Entry"** button
2. Fill in:
   - **Category**: Group name (e.g., "watchlist", "config", "notes")
   - **Key**: Unique identifier within the category (e.g., "AAPL", "thermostat_temp")
   - **Value**: The data to store — plain text or a JSON object
3. Click **"Save"**

### Editing Entries
1. Click on an entry to open it
2. Modify the fields
3. Click **"Save"**

### Deleting Entries
1. Click on an entry to open it
2. Click **"Delete"**
3. Confirm the deletion

## Using the REST API

The REST API is available at `/api/` and is designed for easy use with `curl` and scripts.

### List All Entries

```bash
curl http://localhost:8080/api/entries
```

### Filter by Category

```bash
curl "http://localhost:8080/api/entries?category=watchlist"
```

### Search Entries

```bash
curl "http://localhost:8080/api/entries?search=apple"
```

### Create an Entry

```bash
curl -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."}'
```

### Get an Entry

You can look up entries by numeric ID **or by key**:

```bash
# By numeric ID
curl http://localhost:8080/api/entries/1

# By key — much easier to remember and script
curl http://localhost:8080/api/entries/AAPL
curl http://localhost:8080/api/entries/thermostat_temp
```

### Update an Entry

```bash
# By numeric ID
curl -X PUT http://localhost:8080/api/entries/1 \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'

# By key
curl -X PUT http://localhost:8080/api/entries/AAPL \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'
```

### Delete an Entry

```bash
# By numeric ID
curl -X DELETE http://localhost:8080/api/entries/1

# By key
curl -X DELETE http://localhost:8080/api/entries/AAPL
```

### Get All Categories

```bash
curl http://localhost:8080/api/categories
```

### Health Check

```bash
curl http://localhost:8080/api/health
```

## Working with JSON Values

The `value` field supports both plain text and structured JSON data.

### Plain Text Values

When you store a plain string, the API wraps it in a `{"data": "..."}` envelope
so that the `value` field is always valid JSON:

```bash
$ curl -X POST http://localhost:8080/api/entries \
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
curl -s http://localhost:8080/api/entries/city | jq '.value.data'
# → "San Jose"
```

### JSON Object / Array Values

When you store a JSON object or array, it is embedded directly in the response
without any wrapping:

```bash
$ curl -X POST http://localhost:8080/api/entries \
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
LAT=$(curl -s http://localhost:8080/api/entries/location | jq '.value.lat')
```

### Updating to a JSON Value

```bash
curl -X PUT http://localhost:8080/api/entries/location \
  -H "Content-Type: application/json" \
  -d '{"value": {"lat": 37.77, "lon": -122.41}}'
```

## Import and Export

### Export via GUI
1. Go to the **Settings** page
2. Click **"Export Data"**
3. A JSON file will download automatically

### Export via API

```bash
# Save to file
curl http://localhost:8080/api/export -o homeapi-backup.json

# Pretty print
curl http://localhost:8080/api/export | jq .
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
curl -X POST http://localhost:8080/api/import \
  -H "Content-Type: application/json" \
  -d @homeapi-backup.json

# Replace mode - overwrite existing entries
curl -X POST http://localhost:8080/api/import \
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
| `HOMEAPI_PORT` | `8080` | HTTP server port |
| `HOMEAPI_DB_PATH` | `~/.homeapi/homeapi.db` | Path to SQLite database file |
| `HOMEAPI_LOG_LEVEL` | `info` | Logging level: debug, info, warn, error |

The database file and its directory are created automatically on first run.

## Examples

### Stock Watchlist Script

```bash
#!/bin/bash
# Add stocks to watchlist
for ticker in AAPL GOOGL MSFT AMZN; do
  curl -s -X POST http://localhost:8080/api/entries \
    -H "Content-Type: application/json" \
    -d "{\"category\": \"watchlist\", \"key\": \"$ticker\", \"value\": \"active\"}"
done

# List watchlist keys
curl -s "http://localhost:8080/api/entries?category=watchlist" | jq '.entries[].key'

# Look up a specific stock by key
curl -s http://localhost:8080/api/entries/AAPL | jq '.value.data'
```

### Home Automation Config

```bash
# Set thermostat temperature
curl -s -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "config", "key": "thermostat_temp", "value": "72"}'

# Read it back by key — no ID lookup needed
TEMP=$(curl -s http://localhost:8080/api/entries/thermostat_temp | jq -r '.value.data')
echo "Setting thermostat to $TEMP"
```

### Storing Structured Data

```bash
# Store a JSON config object
curl -s -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "config", "key": "mqtt", "value": {"host": "192.168.1.10", "port": 1883}}'

# Read it back
curl -s http://localhost:8080/api/entries/mqtt | jq '.value.host'
# → "192.168.1.10"
```

### Backup Cron Job

```bash
# Add to crontab: daily backup at midnight
0 0 * * * curl -s http://localhost:8080/api/export > /backups/homeapi-$(date +\%Y\%m\%d).json
```
