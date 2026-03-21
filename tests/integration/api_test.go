package integration

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

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := db.NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	handler := api.NewHandler(store)
	router := api.NewRouter(handler, nil)
	return httptest.NewServer(router)
}

func TestHealthEndpoint(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatalf("GET /api/health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var health models.HealthResponse
	json.NewDecoder(resp.Body).Decode(&health)
	if health.Status != "ok" {
		t.Errorf("status = %q, want %q", health.Status, "ok")
	}
}

func TestCRUDWorkflow(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// 1. Create an entry
	createBody, _ := json.Marshal(map[string]string{
		"category": "watchlist",
		"key":      "AAPL",
		"value":    "Apple Inc.",
	})
	resp, err := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /api/entries: %v", err)
	}
	if resp.StatusCode != 201 {
		t.Fatalf("create status = %d, want 201", resp.StatusCode)
	}
	var created models.Entry
	json.NewDecoder(resp.Body).Decode(&created)
	resp.Body.Close()

	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	// 2. Get the entry
	resp, err = client.Get(ts.URL + fmt.Sprintf("/api/entries/%d", created.ID))
	if err != nil {
		t.Fatalf("GET /api/entries/%d: %v", created.ID, err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("get status = %d, want 200", resp.StatusCode)
	}
	var fetched models.Entry
	json.NewDecoder(resp.Body).Decode(&fetched)
	resp.Body.Close()

	if fetched.Key != "AAPL" {
		t.Errorf("key = %q, want %q", fetched.Key, "AAPL")
	}

	// 3. Update the entry
	updateBody, _ := json.Marshal(map[string]string{"value": "Apple Inc. - Buy"})
	req, _ := http.NewRequest("PUT", ts.URL+fmt.Sprintf("/api/entries/%d", created.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("PUT: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("update status = %d, want 200", resp.StatusCode)
	}
	var updated models.Entry
	json.NewDecoder(resp.Body).Decode(&updated)
	resp.Body.Close()

	if updated.Value != "Apple Inc. - Buy" {
		t.Errorf("value = %q, want %q", updated.Value, "Apple Inc. - Buy")
	}

	// 4. List entries
	resp, _ = client.Get(ts.URL + "/api/entries")
	var listed models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&listed)
	resp.Body.Close()

	if listed.Total != 1 {
		t.Errorf("total = %d, want 1", listed.Total)
	}

	// 5. Delete the entry
	req, _ = http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/entries/%d", created.ID), nil)
	resp, _ = client.Do(req)
	if resp.StatusCode != 204 {
		t.Fatalf("delete status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// 6. Verify deleted
	resp, _ = client.Get(ts.URL + fmt.Sprintf("/api/entries/%d", created.ID))
	if resp.StatusCode != 404 {
		t.Errorf("after delete status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCategoryFiltering(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	entries := []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple"},
		{"category": "watchlist", "key": "GOOGL", "value": "Google"},
		{"category": "config", "key": "temp", "value": "72"},
		{"category": "config", "key": "humidity", "value": "45"},
		{"category": "config", "key": "light", "value": "on"},
	}
	for _, e := range entries {
		body, _ := json.Marshal(e)
		resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		resp.Body.Close()
	}

	// List categories
	resp, _ := client.Get(ts.URL + "/api/categories")
	var cats []models.CategoryInfo
	json.NewDecoder(resp.Body).Decode(&cats)
	resp.Body.Close()

	if len(cats) != 2 {
		t.Fatalf("got %d categories, want 2", len(cats))
	}

	// Filter by category
	resp, _ = client.Get(ts.URL + "/api/entries?category=watchlist")
	var result models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result.Total != 2 {
		t.Errorf("watchlist total = %d, want 2", result.Total)
	}

	resp, _ = client.Get(ts.URL + "/api/entries?category=config")
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result.Total != 3 {
		t.Errorf("config total = %d, want 3", result.Total)
	}
}

func TestSearchEntries(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	entries := []map[string]string{
		{"category": "stocks", "key": "AAPL", "value": "Apple Inc."},
		{"category": "stocks", "key": "GOOGL", "value": "Alphabet Inc."},
		{"category": "notes", "key": "shopping", "value": "Buy apples and oranges"},
	}
	for _, e := range entries {
		body, _ := json.Marshal(e)
		resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		resp.Body.Close()
	}

	resp, _ := client.Get(ts.URL + "/api/entries?search=AAPL")
	var result models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result.Total != 1 {
		t.Errorf("search AAPL total = %d, want 1", result.Total)
	}

	// "apple" matches "Apple Inc." and "Buy apples and oranges"
	resp, _ = client.Get(ts.URL + "/api/entries?search=apple")
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if result.Total != 2 {
		t.Errorf("search apple total = %d, want 2", result.Total)
	}
}

func TestPagination(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	for i := 0; i < 10; i++ {
		body, _ := json.Marshal(map[string]string{
			"category": "test",
			"key":      fmt.Sprintf("key_%02d", i),
			"value":    fmt.Sprintf("val_%d", i),
		})
		resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		resp.Body.Close()
	}

	resp, _ := client.Get(ts.URL + "/api/entries?per_page=3&page=1")
	var result models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if len(result.Entries) != 3 {
		t.Errorf("page 1 entries = %d, want 3", len(result.Entries))
	}
	if result.Total != 10 {
		t.Errorf("total = %d, want 10", result.Total)
	}
	if result.TotalPages != 4 {
		t.Errorf("total_pages = %d, want 4", result.TotalPages)
	}

	resp, _ = client.Get(ts.URL + "/api/entries?per_page=3&page=4")
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()

	if len(result.Entries) != 1 {
		t.Errorf("page 4 entries = %d, want 1", len(result.Entries))
	}
}

func TestDuplicateEntryConflict(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	body, _ := json.Marshal(map[string]string{"category": "test", "key": "dup", "value": "v1"})

	resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
	if resp.StatusCode != 201 {
		t.Fatalf("first create: %d", resp.StatusCode)
	}
	resp.Body.Close()

	resp, _ = client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
	if resp.StatusCode != 409 {
		t.Errorf("duplicate: status = %d, want 409", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestCORSHeaders(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	req, _ := http.NewRequest("OPTIONS", ts.URL+"/api/entries", nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("OPTIONS: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Error("expected CORS Allow-Origin header")
	}
	if resp.Header.Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected CORS Allow-Methods header")
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	client := ts.Client()

	// Create entries
	for _, e := range []map[string]string{
		{"category": "test", "key": "k1", "value": "v1"},
		{"category": "test", "key": "k2", "value": "v2"},
		{"category": "other", "key": "k3", "value": "v3"},
	} {
		body, _ := json.Marshal(e)
		resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		resp.Body.Close()
	}

	// Export
	resp, _ := client.Get(ts.URL + "/api/export")
	var exportData models.ExportData
	json.NewDecoder(resp.Body).Decode(&exportData)
	resp.Body.Close()

	if len(exportData.Entries) != 3 {
		t.Fatalf("export entries = %d, want 3", len(exportData.Entries))
	}

	// Delete all
	for _, e := range exportData.Entries {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/entries/%d", ts.URL, e.ID), nil)
		resp, _ := client.Do(req)
		resp.Body.Close()
	}

	// Verify empty
	resp, _ = client.Get(ts.URL + "/api/entries")
	var listed models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&listed)
	resp.Body.Close()
	if listed.Total != 0 {
		t.Fatalf("after delete total = %d, want 0", listed.Total)
	}

	// Import back
	importReq := models.ImportRequest{
		Entries: exportData.Entries,
		Mode:    "merge",
	}
	importBody, _ := json.Marshal(importReq)
	resp, _ = client.Post(ts.URL+"/api/import", "application/json", bytes.NewReader(importBody))
	var importResult models.ImportResult
	json.NewDecoder(resp.Body).Decode(&importResult)
	resp.Body.Close()

	if importResult.Imported != 3 {
		t.Errorf("imported = %d, want 3", importResult.Imported)
	}

	// Verify restored
	resp, _ = client.Get(ts.URL + "/api/entries")
	json.NewDecoder(resp.Body).Decode(&listed)
	resp.Body.Close()
	if listed.Total != 3 {
		t.Errorf("restored total = %d, want 3", listed.Total)
	}
}
