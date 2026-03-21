# HomeAPI - User Guide

## Table of Contents
1. [Getting Started](#getting-started)
2. [Using the Web Interface](#using-the-web-interface)
3. [Using the REST API](#using-the-rest-api)
4. [Import and Export](#import-and-export)
5. [Configuration](#configuration)
6. [Examples](#examples)

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
   - **Value**: The data to store (e.g., "Apple Inc.", "72")
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

### Update an Entry

```bash
curl -X PUT http://localhost:8080/api/entries/1 \
  -H "Content-Type: application/json" \
  -d '{"value": "Apple Inc. - Buy"}'
```

### Delete an Entry

```bash
curl -X DELETE http://localhost:8080/api/entries/1
```

### Get All Categories

```bash
curl http://localhost:8080/api/categories
```

### Health Check

```bash
curl http://localhost:8080/api/health
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
      "category": "watchlist",
      "key": "AAPL",
      "value": "Apple Inc.",
      "created_at": "2024-01-10T08:00:00Z",
      "updated_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

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

# List watchlist
curl -s "http://localhost:8080/api/entries?category=watchlist" | jq '.entries[].key'
```

### Home Automation Config

```bash
# Set thermostat temperature
curl -s -X POST http://localhost:8080/api/entries \
  -H "Content-Type: application/json" \
  -d '{"category": "config", "key": "thermostat_temp", "value": "72"}'

# Read it from another script
TEMP=$(curl -s "http://localhost:8080/api/entries?category=config&search=thermostat" | jq -r '.entries[0].value')
echo "Setting thermostat to $TEMP"
```

### Backup Cron Job

```bash
# Add to crontab: daily backup at midnight
0 0 * * * curl -s http://localhost:8080/api/export > /backups/homeapi-$(date +\%Y\%m\%d).json
```
