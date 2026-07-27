package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

// seedBulkEntries creates the fixture entries used by the bulk delete tests.
func seedBulkEntries(t *testing.T, ts *httptest.Server) {
	t.Helper()
	for _, e := range []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."},
		{"category": "watchlist", "key": "GOOGL", "value": "Google LLC"},
		{"category": "watchlist", "key": "MSFT", "value": "Microsoft Corp."},
		{"category": "config", "key": "thermostat", "value": "72"},
		{"category": "notes", "key": "groceries", "value": "milk, eggs"},
	} {
		body, _ := json.Marshal(e)
		resp, err := ts.Client().Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("POST /api/entries: %v", err)
		}
		if resp.StatusCode != 201 {
			t.Fatalf("seed %s: status = %d", e["key"], resp.StatusCode)
		}
		resp.Body.Close()
	}
}

// doBulkDelete sends DELETE /api/entries through the router and decodes the result.
func doBulkDelete(t *testing.T, ts *httptest.Server, query string, body io.Reader) (int, models.BulkDeleteResult) {
	t.Helper()
	req, err := http.NewRequest("DELETE", ts.URL+"/api/entries"+query, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/entries%s: %v", query, err)
	}
	defer resp.Body.Close()

	var result models.BulkDeleteResult
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func totalEntries(t *testing.T, ts *httptest.Server) int {
	t.Helper()
	resp, err := ts.Client().Get(ts.URL + "/api/entries?per_page=200")
	if err != nil {
		t.Fatalf("GET /api/entries: %v", err)
	}
	defer resp.Body.Close()

	var list models.PaginatedEntries
	json.NewDecoder(resp.Body).Decode(&list)
	return list.Total
}

func TestBulkDeleteByKeys(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	status, result := doBulkDelete(t, ts, "?keys=AAPL,GOOGL", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if len(result.Entries) != 2 {
		t.Errorf("entries = %d, want 2", len(result.Entries))
	}
	if got := totalEntries(t, ts); got != 3 {
		t.Errorf("remaining = %d, want 3", got)
	}

	// The deleted entries are really gone.
	resp, _ := ts.Client().Get(ts.URL + "/api/entries/AAPL")
	if resp.StatusCode != 404 {
		t.Errorf("GET deleted key status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestBulkDeleteByCategory(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	status, result := doBulkDelete(t, ts, "?category=watchlist", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 3 {
		t.Errorf("deleted = %d, want 3", result.Deleted)
	}
	if got := totalEntries(t, ts); got != 2 {
		t.Errorf("remaining = %d, want 2", got)
	}

	// The emptied category disappears from the category listing.
	resp, _ := ts.Client().Get(ts.URL + "/api/categories")
	var categories []models.CategoryInfo
	json.NewDecoder(resp.Body).Decode(&categories)
	resp.Body.Close()
	for _, c := range categories {
		if c.Name == "watchlist" {
			t.Errorf("category watchlist still present with count %d", c.Count)
		}
	}
}

func TestBulkDeleteBySearch(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	// Search matches key or value, so "Google" hits the GOOGL entry's value.
	status, result := doBulkDelete(t, ts, "?search=Google", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", result.Deleted)
	}
	if len(result.Entries) != 1 || result.Entries[0].Key != "GOOGL" {
		t.Errorf("entries = %+v, want GOOGL", result.Entries)
	}
}

func TestBulkDeleteWithJSONBody(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	body, _ := json.Marshal(map[string]interface{}{"keys": []string{"AAPL", "thermostat"}})
	status, result := doBulkDelete(t, ts, "", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if got := totalEntries(t, ts); got != 3 {
		t.Errorf("remaining = %d, want 3", got)
	}
}

func TestBulkDeleteDryRun(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	status, result := doBulkDelete(t, ts, "?category=watchlist&dry_run=true", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if !result.DryRun || result.Deleted != 0 || result.Matched != 3 {
		t.Errorf("result = %+v, want dry run with 3 matched and 0 deleted", result)
	}
	if got := totalEntries(t, ts); got != 5 {
		t.Errorf("remaining = %d, want 5", got)
	}
}

func TestBulkDeleteRequiresSelector(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	req, _ := http.NewRequest("DELETE", ts.URL+"/api/entries", nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/entries: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 400 {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	var errResp models.ErrorResponse
	json.NewDecoder(resp.Body).Decode(&errResp)
	if errResp.Code != "VALIDATION_ERROR" {
		t.Errorf("code = %q, want VALIDATION_ERROR", errResp.Code)
	}
	if got := totalEntries(t, ts); got != 5 {
		t.Errorf("remaining = %d, want 5 (nothing deleted)", got)
	}
}

func TestBulkDeleteAll(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	status, result := doBulkDelete(t, ts, "?all=true", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 5 {
		t.Errorf("deleted = %d, want 5", result.Deleted)
	}
	if got := totalEntries(t, ts); got != 0 {
		t.Errorf("remaining = %d, want 0", got)
	}
}

// TestSingleEntryDeleteStillWorks guards the pre-existing per-entry DELETE
// routes against regressions from the bulk delete route.
func TestSingleEntryDeleteStillWorks(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	seedBulkEntries(t, ts)

	// By key.
	req, _ := http.NewRequest("DELETE", ts.URL+"/api/entries/AAPL", nil)
	resp, err := ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE by key: %v", err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("delete by key status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// By numeric ID.
	resp, err = ts.Client().Get(ts.URL + "/api/entries/GOOGL")
	if err != nil {
		t.Fatalf("GET by key: %v", err)
	}
	var entry apiEntry
	json.NewDecoder(resp.Body).Decode(&entry)
	resp.Body.Close()

	req, _ = http.NewRequest("DELETE", ts.URL+fmt.Sprintf("/api/entries/%d", entry.ID), nil)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE by id: %v", err)
	}
	if resp.StatusCode != 204 {
		t.Errorf("delete by id status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()

	// Deleting a missing entry still 404s rather than falling through to bulk delete.
	req, _ = http.NewRequest("DELETE", ts.URL+"/api/entries/NOPE", nil)
	resp, err = ts.Client().Do(req)
	if err != nil {
		t.Fatalf("DELETE missing: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("delete missing status = %d, want 404", resp.StatusCode)
	}
	resp.Body.Close()

	if got := totalEntries(t, ts); got != 3 {
		t.Errorf("remaining = %d, want 3", got)
	}
}
