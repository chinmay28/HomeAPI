package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/chinmay28/homeapi/internal/db"
	"github.com/chinmay28/homeapi/internal/models"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	store, err := db.NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return NewHandler(store)
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	h.Health(w, req)

	if w.Code != 200 {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp models.HealthResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.Version != Version {
		t.Errorf("version = %q, want %q", resp.Version, Version)
	}
}

func TestCreateAndListEntries(t *testing.T) {
	h := newTestHandler(t)

	// Create an entry with a plain string value
	body, _ := json.Marshal(map[string]string{
		"category": "watchlist",
		"key":      "AAPL",
		"value":    "Apple Inc.",
	})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)

	if w.Code != 201 {
		t.Fatalf("create status = %d, want 201, body: %s", w.Code, w.Body.String())
	}

	var created entryResponse
	json.NewDecoder(w.Body).Decode(&created)
	if created.Key != "AAPL" {
		t.Errorf("key = %q, want %q", created.Key, "AAPL")
	}
	// Plain string value should be wrapped as {"data": "Apple Inc."}
	var wrapped map[string]string
	json.Unmarshal(created.Value, &wrapped)
	if wrapped["data"] != "Apple Inc." {
		t.Errorf("value.data = %q, want %q", wrapped["data"], "Apple Inc.")
	}

	// List entries
	req = httptest.NewRequest("GET", "/api/entries", nil)
	w = httptest.NewRecorder()
	h.ListEntries(w, req)

	if w.Code != 200 {
		t.Fatalf("list status = %d, want 200", w.Code)
	}

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func TestCreateEntry_InvalidBody(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateEntry_MissingKey(t *testing.T) {
	h := newTestHandler(t)
	body, _ := json.Marshal(map[string]string{"category": "test", "value": "v"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestCreateEntry_Duplicate(t *testing.T) {
	h := newTestHandler(t)
	body, _ := json.Marshal(map[string]string{"category": "test", "key": "k1", "value": "v"})

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 201 {
		t.Fatalf("first create: status = %d", w.Code)
	}

	req = httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 409 {
		t.Errorf("duplicate: status = %d, want 409", w.Code)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/entries/999", nil)
	w := httptest.NewRecorder()
	h.GetEntry(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestGetEntry_ByKey(t *testing.T) {
	h := newTestHandler(t)

	// Create an entry
	body, _ := json.Marshal(map[string]string{"category": "default", "key": "city", "value": "San Jose"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 201 {
		t.Fatalf("create: status = %d", w.Code)
	}

	// Look it up by key
	req = httptest.NewRequest("GET", "/api/entries/city", nil)
	w = httptest.NewRecorder()
	h.GetEntry(w, req)

	if w.Code != 200 {
		t.Fatalf("get by key: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var resp entryResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Key != "city" {
		t.Errorf("key = %q, want %q", resp.Key, "city")
	}
	var wrapped map[string]string
	json.Unmarshal(resp.Value, &wrapped)
	if wrapped["data"] != "San Jose" {
		t.Errorf("value.data = %q, want %q", wrapped["data"], "San Jose")
	}
}

func TestGetEntry_KeyNotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/entries/nonexistent", nil)
	w := httptest.NewRecorder()
	h.GetEntry(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestUpdateEntry(t *testing.T) {
	h := newTestHandler(t)

	// Create
	body, _ := json.Marshal(map[string]string{"category": "config", "key": "temp", "value": "72"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	var created entryResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update by numeric ID
	updateBody, _ := json.Marshal(map[string]string{"value": "75"})
	req = httptest.NewRequest("PUT", "/api/entries/"+itoa(created.ID), bytes.NewReader(updateBody))
	w = httptest.NewRecorder()
	h.UpdateEntry(w, req)

	if w.Code != 200 {
		t.Fatalf("update status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var updated entryResponse
	json.NewDecoder(w.Body).Decode(&updated)
	var wrapped map[string]string
	json.Unmarshal(updated.Value, &wrapped)
	if wrapped["data"] != "75" {
		t.Errorf("value.data = %q, want %q", wrapped["data"], "75")
	}
}

func TestUpdateEntry_ByKey(t *testing.T) {
	h := newTestHandler(t)

	// Create
	body, _ := json.Marshal(map[string]string{"category": "config", "key": "mode", "value": "off"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 201 {
		t.Fatalf("create: status = %d", w.Code)
	}

	// Update by key
	updateBody, _ := json.Marshal(map[string]string{"value": "on"})
	req = httptest.NewRequest("PUT", "/api/entries/mode", bytes.NewReader(updateBody))
	w = httptest.NewRecorder()
	h.UpdateEntry(w, req)

	if w.Code != 200 {
		t.Fatalf("update by key: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var updated entryResponse
	json.NewDecoder(w.Body).Decode(&updated)
	var wrapped map[string]string
	json.Unmarshal(updated.Value, &wrapped)
	if wrapped["data"] != "on" {
		t.Errorf("value.data = %q, want %q", wrapped["data"], "on")
	}
}

func TestUpdateEntry_JSONValue(t *testing.T) {
	h := newTestHandler(t)

	// Create with plain value
	body, _ := json.Marshal(map[string]string{"category": "config", "key": "location", "value": "home"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	var created entryResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Update with a JSON object value
	updateBody := []byte(`{"value": {"lat": 37.3, "lon": -121.9}}`)
	req = httptest.NewRequest("PUT", "/api/entries/location", bytes.NewReader(updateBody))
	w = httptest.NewRecorder()
	h.UpdateEntry(w, req)

	if w.Code != 200 {
		t.Fatalf("update with JSON value: status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var updated entryResponse
	json.NewDecoder(w.Body).Decode(&updated)
	var coords map[string]float64
	if err := json.Unmarshal(updated.Value, &coords); err != nil {
		t.Fatalf("value is not a JSON object: %s", updated.Value)
	}
	if coords["lat"] != 37.3 {
		t.Errorf("lat = %v, want 37.3", coords["lat"])
	}
}

func TestDeleteEntry(t *testing.T) {
	h := newTestHandler(t)

	// Create
	body, _ := json.Marshal(map[string]string{"category": "test", "key": "del", "value": "v"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	var created entryResponse
	json.NewDecoder(w.Body).Decode(&created)

	// Delete by numeric ID
	req = httptest.NewRequest("DELETE", "/api/entries/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	h.DeleteEntry(w, req)

	if w.Code != 204 {
		t.Errorf("delete status = %d, want 204", w.Code)
	}

	// Verify gone
	req = httptest.NewRequest("GET", "/api/entries/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	h.GetEntry(w, req)
	if w.Code != 404 {
		t.Errorf("after delete: status = %d, want 404", w.Code)
	}
}

func TestDeleteEntry_ByKey(t *testing.T) {
	h := newTestHandler(t)

	body, _ := json.Marshal(map[string]string{"category": "test", "key": "todelete", "value": "v"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 201 {
		t.Fatalf("create: status = %d", w.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/entries/todelete", nil)
	w = httptest.NewRecorder()
	h.DeleteEntry(w, req)
	if w.Code != 204 {
		t.Errorf("delete by key: status = %d, want 204", w.Code)
	}

	req = httptest.NewRequest("GET", "/api/entries/todelete", nil)
	w = httptest.NewRecorder()
	h.GetEntry(w, req)
	if w.Code != 404 {
		t.Errorf("after delete by key: status = %d, want 404", w.Code)
	}
}

func TestDeleteEntry_NotFound(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("DELETE", "/api/entries/999", nil)
	w := httptest.NewRecorder()
	h.DeleteEntry(w, req)

	if w.Code != 404 {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestListCategories(t *testing.T) {
	h := newTestHandler(t)

	for _, e := range []map[string]string{
		{"category": "a", "key": "k1", "value": "v"},
		{"category": "a", "key": "k2", "value": "v"},
		{"category": "b", "key": "k1", "value": "v"},
	} {
		body, _ := json.Marshal(e)
		req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.CreateEntry(w, req)
	}

	req := httptest.NewRequest("GET", "/api/categories", nil)
	w := httptest.NewRecorder()
	h.ListCategories(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var cats []models.CategoryInfo
	json.NewDecoder(w.Body).Decode(&cats)
	if len(cats) != 2 {
		t.Errorf("got %d categories, want 2", len(cats))
	}
}

func TestExportData(t *testing.T) {
	h := newTestHandler(t)

	body, _ := json.Marshal(map[string]string{"category": "test", "key": "k1", "value": "v1"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)

	req = httptest.NewRequest("GET", "/api/export", nil)
	w = httptest.NewRecorder()
	h.ExportData(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var export models.ExportData
	json.NewDecoder(w.Body).Decode(&export)
	if len(export.Entries) != 1 {
		t.Errorf("entries = %d, want 1", len(export.Entries))
	}
	if export.Version != "1" {
		t.Errorf("version = %q, want %q", export.Version, "1")
	}
}

func TestImportData(t *testing.T) {
	h := newTestHandler(t)

	importReq := models.ImportRequest{
		Entries: []models.Entry{
			{Category: "test", Key: "k1", Value: "v1"},
			{Category: "test", Key: "k2", Value: "v2"},
		},
		Mode: "merge",
	}
	body, _ := json.Marshal(importReq)
	req := httptest.NewRequest("POST", "/api/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportData(w, req)

	if w.Code != 200 {
		t.Fatalf("status = %d, want 200, body: %s", w.Code, w.Body.String())
	}

	var result models.ImportResult
	json.NewDecoder(w.Body).Decode(&result)
	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}
}

func TestImportData_InvalidBody(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("POST", "/api/import", bytes.NewReader([]byte("invalid")))
	w := httptest.NewRecorder()
	h.ImportData(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestImportData_EmptyEntries(t *testing.T) {
	h := newTestHandler(t)
	body, _ := json.Marshal(models.ImportRequest{Entries: []models.Entry{}})
	req := httptest.NewRequest("POST", "/api/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportData(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestImportData_InvalidMode(t *testing.T) {
	h := newTestHandler(t)
	body, _ := json.Marshal(map[string]interface{}{
		"entries": []map[string]string{{"key": "k1"}},
		"mode":    "invalid",
	})
	req := httptest.NewRequest("POST", "/api/import", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.ImportData(w, req)

	if w.Code != 400 {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestListEntries_WithFilters(t *testing.T) {
	h := newTestHandler(t)

	// Create entries in different categories
	for _, e := range []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple"},
		{"category": "watchlist", "key": "GOOGL", "value": "Google"},
		{"category": "config", "key": "temp", "value": "72"},
	} {
		body, _ := json.Marshal(e)
		req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.CreateEntry(w, req)
	}

	// Filter by category
	req := httptest.NewRequest("GET", "/api/entries?category=watchlist", nil)
	w := httptest.NewRecorder()
	h.ListEntries(w, req)

	var result struct {
		Total int `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&result)
	if result.Total != 2 {
		t.Errorf("total = %d, want 2", result.Total)
	}

	// Search
	req = httptest.NewRequest("GET", "/api/entries?search=Apple", nil)
	w = httptest.NewRecorder()
	h.ListEntries(w, req)

	json.NewDecoder(w.Body).Decode(&result)
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}
}

func itoa(n int64) string {
	return fmt.Sprintf("%d", n)
}
