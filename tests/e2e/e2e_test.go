package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chinmay28/homeapi/internal/api"
	"github.com/chinmay28/homeapi/internal/db"
	"github.com/chinmay28/homeapi/internal/models"
)

func startServer(t *testing.T) (*httptest.Server, *http.Client) {
	t.Helper()
	store, err := db.NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	handler := api.NewHandler(store)
	router := api.NewRouter(handler, nil)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)
	return ts, ts.Client()
}

func postEntry(t *testing.T, client *http.Client, url string, entry map[string]string) models.Entry {
	t.Helper()
	body, _ := json.Marshal(entry)
	resp, err := client.Post(url+"/api/entries", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /api/entries: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		t.Fatalf("create failed: status %d", resp.StatusCode)
	}
	var e models.Entry
	json.NewDecoder(resp.Body).Decode(&e)
	return e
}

func TestFullWorkflow(t *testing.T) {
	ts, client := startServer(t)

	// Step 1: Health check
	resp, _ := client.Get(ts.URL + "/api/health")
	var health models.HealthResponse
	json.NewDecoder(resp.Body).Decode(&health)
	resp.Body.Close()
	if health.Status != "ok" {
		t.Fatalf("health check failed: %q", health.Status)
	}

	// Step 2: Create stock watchlist entries
	stocks := []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."},
		{"category": "watchlist", "key": "GOOGL", "value": "Alphabet Inc."},
		{"category": "watchlist", "key": "MSFT", "value": "Microsoft Corp."},
		{"category": "watchlist", "key": "AMZN", "value": "Amazon.com Inc."},
	}
	var stockIDs []int64
	for _, s := range stocks {
		e := postEntry(t, client, ts.URL, s)
		stockIDs = append(stockIDs, e.ID)
	}

	// Step 3: Create config entries
	configs := []map[string]string{
		{"category": "config", "key": "thermostat_temp", "value": "72"},
		{"category": "config", "key": "light_brightness", "value": "80"},
		{"category": "config", "key": "alarm_time", "value": "07:00"},
	}
	for _, c := range configs {
		postEntry(t, client, ts.URL, c)
	}

	// Step 4: Verify categories
	resp, _ = client.Get(ts.URL + "/api/categories")
	var cats []models.CategoryInfo
	json.NewDecoder(resp.Body).Decode(&cats)
	resp.Body.Close()

	if len(cats) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(cats))
	}
	catMap := map[string]int{}
	for _, c := range cats {
		catMap[c.Name] = c.Count
	}
	if catMap["watchlist"] != 4 {
		t.Errorf("watchlist count = %d, want 4", catMap["watchlist"])
	}
	if catMap["config"] != 3 {
		t.Errorf("config count = %d, want 3", catMap["config"])
	}

	// Step 5: Search for "Apple"
	resp, _ = client.Get(ts.URL + "/api/entries?search=Apple")
	var searchResult models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&searchResult)
	resp.Body.Close()
	if searchResult.Total != 1 {
		t.Errorf("search Apple total = %d, want 1", searchResult.Total)
	}

	// Step 6: Update a stock entry
	updateBody, _ := json.Marshal(map[string]string{"value": "Apple Inc. - Strong Buy"})
	req, _ := http.NewRequest("PUT", fmt.Sprintf("%s/api/entries/%d", ts.URL, stockIDs[0]), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	var updatedEntry models.Entry
	json.NewDecoder(resp.Body).Decode(&updatedEntry)
	resp.Body.Close()
	if updatedEntry.Value != "Apple Inc. - Strong Buy" {
		t.Errorf("updated value = %q", updatedEntry.Value)
	}

	// Step 7: Export all data
	resp, _ = client.Get(ts.URL + "/api/export")
	var exportData models.ExportData
	json.NewDecoder(resp.Body).Decode(&exportData)
	resp.Body.Close()

	if len(exportData.Entries) != 7 {
		t.Fatalf("export entries = %d, want 7", len(exportData.Entries))
	}

	// Step 8: Delete all stock entries
	for _, id := range stockIDs {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/entries/%d", ts.URL, id), nil)
		resp, _ := client.Do(req)
		if resp.StatusCode != 204 {
			t.Errorf("delete %d: status %d", id, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// Verify only config entries remain
	resp, _ = client.Get(ts.URL + "/api/entries")
	var afterDelete models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&afterDelete)
	resp.Body.Close()
	if afterDelete.Total != 3 {
		t.Errorf("after delete total = %d, want 3", afterDelete.Total)
	}

	// Step 9: Re-import exported data (merge mode)
	importReq := models.ImportRequest{
		Entries: exportData.Entries,
		Mode:    "merge",
	}
	importBody, _ := json.Marshal(importReq)
	resp, _ = client.Post(ts.URL+"/api/import", "application/json", bytes.NewReader(importBody))
	var importResult models.ImportResult
	json.NewDecoder(resp.Body).Decode(&importResult)
	resp.Body.Close()

	if importResult.Imported != 4 {
		t.Errorf("imported = %d, want 4 (stocks)", importResult.Imported)
	}
	if importResult.Skipped != 3 {
		t.Errorf("skipped = %d, want 3 (configs)", importResult.Skipped)
	}

	// Step 10: Verify all 7 entries restored
	resp, _ = client.Get(ts.URL + "/api/entries?per_page=100")
	var finalList models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&finalList)
	resp.Body.Close()
	if finalList.Total != 7 {
		t.Errorf("final total = %d, want 7", finalList.Total)
	}
}

func TestImportReplace(t *testing.T) {
	ts, client := startServer(t)

	postEntry(t, client, ts.URL, map[string]string{
		"category": "config",
		"key":      "temp",
		"value":    "72",
	})

	importReq := models.ImportRequest{
		Entries: []models.Entry{
			{Category: "config", Key: "temp", Value: "68"},
			{Category: "config", Key: "humidity", Value: "45"},
		},
		Mode: "replace",
	}
	body, _ := json.Marshal(importReq)
	resp, _ := client.Post(ts.URL+"/api/import", "application/json", bytes.NewReader(body))
	var result models.ImportResult
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}

	// Verify temp was overwritten
	resp, _ = client.Get(ts.URL + "/api/entries?search=temp")
	var entries models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&entries)
	resp.Body.Close()

	if len(entries.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries.Entries))
	}
	if entries.Entries[0].Value != "68" {
		t.Errorf("value = %q, want %q", entries.Entries[0].Value, "68")
	}
}

func TestScriptLikeUsage(t *testing.T) {
	ts, client := startServer(t)

	// Script: Stock price updater
	tickers := []string{"AAPL", "GOOGL", "TSLA", "NVDA", "META"}
	for _, ticker := range tickers {
		body, _ := json.Marshal(map[string]string{
			"category": "stock_prices",
			"key":      ticker,
			"value":    "150.00",
		})
		resp, err := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("create %s: %v", ticker, err)
		}
		if resp.StatusCode != 201 {
			t.Fatalf("create %s: status %d", ticker, resp.StatusCode)
		}
		resp.Body.Close()
	}

	// Script: Config reader
	postEntry(t, client, ts.URL, map[string]string{
		"category": "config",
		"key":      "alert_threshold",
		"value":    "10",
	})

	resp, _ := client.Get(ts.URL + "/api/entries?category=config&search=alert_threshold")
	var result models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if len(result.Entries) != 1 {
		t.Fatalf("expected 1 config entry, got %d", len(result.Entries))
	}
	if result.Entries[0].Value != "10" {
		t.Errorf("config value = %q, want %q", result.Entries[0].Value, "10")
	}

	// Export for backup
	resp, _ = client.Get(ts.URL + "/api/export")
	var export models.ExportData
	json.NewDecoder(resp.Body).Decode(&export)
	resp.Body.Close()

	if len(export.Entries) != 6 {
		t.Errorf("export entries = %d, want 6", len(export.Entries))
	}
}

func TestMethodNotAllowed(t *testing.T) {
	ts, client := startServer(t)

	tests := []struct {
		method string
		path   string
	}{
		{"DELETE", "/api/entries"},
		{"POST", "/api/categories"},
		{"PUT", "/api/export"},
		{"GET", "/api/import"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("%s %s: %v", tt.method, tt.path, err)
			}
			resp.Body.Close()
			if resp.StatusCode != 405 {
				t.Errorf("status = %d, want 405", resp.StatusCode)
			}
		})
	}
}

func TestNotFoundAPIEndpoint(t *testing.T) {
	ts, client := startServer(t)

	resp, err := client.Get(ts.URL + "/api/nonexistent")
	if err != nil {
		t.Fatalf("GET /api/nonexistent: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
