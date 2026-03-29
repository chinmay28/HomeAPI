package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

// TestOldExportFormatImport verifies that backup files created by older versions
// of HomeAPI (where "value" is a plain JSON string, not a wrapped object) can
// still be imported without errors. This must never break.
func TestOldExportFormatImport(t *testing.T) {
	ts, client := startServer(t)

	// This is what an export file looked like before the JSON value change.
	// The "value" field is a plain JSON string — not {"data": "..."}.
	oldExport := `{
		"version": "1",
		"exported_at": "2024-01-01T00:00:00Z",
		"entries": [
			{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."},
			{"category": "config",    "key": "temp", "value": "72"},
			{"category": "notes",     "key": "todo", "value": "buy groceries"}
		]
	}`

	resp, err := client.Post(ts.URL+"/api/import", "application/json", bytes.NewReader([]byte(oldExport)))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("import status = %d, want 200 — old export files must still import cleanly", resp.StatusCode)
	}

	var result models.ImportResult
	json.NewDecoder(resp.Body).Decode(&result)
	if result.Imported != 3 {
		t.Errorf("imported = %d, want 3", result.Imported)
	}
	if result.Errors != 0 {
		t.Errorf("errors = %d, want 0", result.Errors)
	}

	// Entries imported from old format must be readable via the current API.
	resp2, _ := client.Get(ts.URL + "/api/entries/AAPL")
	defer resp2.Body.Close()
	if resp2.StatusCode != 200 {
		t.Fatalf("get imported entry: status = %d, want 200", resp2.StatusCode)
	}
	var entry apiEntry
	json.NewDecoder(resp2.Body).Decode(&entry)
	if got := entryValue(entry.Value); got != "Apple Inc." {
		t.Errorf("value = %q, want %q", got, "Apple Inc.")
	}
}

// TestExportFormatStability verifies that the export format keeps all required
// top-level fields and that entry values are exported as plain strings (not the
// JSON-wrapped form used by regular API endpoints). Old backup files rely on this.
func TestExportFormatStability(t *testing.T) {
	ts, client := startServer(t)

	postEntry(t, client, ts.URL, map[string]string{"category": "test", "key": "k1", "value": "hello"})

	resp, _ := client.Get(ts.URL + "/api/export")
	defer resp.Body.Close()

	var raw map[string]json.RawMessage
	json.NewDecoder(resp.Body).Decode(&raw)

	// Required top-level fields must be present.
	for _, field := range []string{"version", "exported_at", "entries"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("export missing required top-level field %q", field)
		}
	}

	// Version must be "1" (or a value clients can check for compatibility).
	var version string
	json.Unmarshal(raw["version"], &version)
	if version != "1" {
		t.Errorf("export version = %q, want %q — version must not change without a migration path", version, "1")
	}

	// Each entry must have all required fields.
	var entries []map[string]json.RawMessage
	json.Unmarshal(raw["entries"], &entries)
	if len(entries) != 1 {
		t.Fatalf("export entries = %d, want 1", len(entries))
	}
	for _, field := range []string{"id", "category", "key", "value", "created_at", "updated_at"} {
		if _, ok := entries[0][field]; !ok {
			t.Errorf("export entry missing required field %q", field)
		}
	}

	// The "value" field in exports must be a plain JSON string, not a wrapped
	// object. This is critical for import round-trips and backup compatibility.
	var exportedValue string
	if err := json.Unmarshal(entries[0]["value"], &exportedValue); err != nil {
		t.Errorf("export entry value must be a plain JSON string (not a wrapped object), got: %s — "+
			"this would break re-import of backup files", entries[0]["value"])
	}
	if exportedValue != "hello" {
		t.Errorf("exported value = %q, want %q", exportedValue, "hello")
	}
}

// TestExportImportRoundTripPreservesValues verifies that exporting and re-importing
// data preserves all values exactly, including JSON objects stored as values.
func TestExportImportRoundTripPreservesValues(t *testing.T) {
	ts, client := startServer(t)

	// Create entries with mixed value types.
	entries := []struct {
		key      string
		bodyJSON string
	}{
		{"plain", `{"category":"test","key":"plain","value":"hello"}`},
		{"jsonobj", `{"category":"test","key":"jsonobj","value":{"x":1,"y":"two"}}`},
		{"arr", `{"category":"test","key":"arr","value":["a","b","c"]}`},
		{"empty", `{"category":"test","key":"empty","value":""}`},
	}
	for _, e := range entries {
		resp, _ := client.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader([]byte(e.bodyJSON)))
		resp.Body.Close()
	}

	// Export.
	resp, _ := client.Get(ts.URL + "/api/export")
	var export models.ExportData
	json.NewDecoder(resp.Body).Decode(&export)
	resp.Body.Close()
	if len(export.Entries) != 4 {
		t.Fatalf("export entries = %d, want 4", len(export.Entries))
	}

	// Delete all.
	for _, e := range export.Entries {
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/api/entries/%d", ts.URL, e.ID), nil)
		resp, _ := client.Do(req)
		resp.Body.Close()
	}

	// Re-import from the export.
	importBody, _ := json.Marshal(models.ImportRequest{Entries: export.Entries, Mode: "merge"})
	resp, _ = client.Post(ts.URL+"/api/import", "application/json", bytes.NewReader(importBody))
	var result models.ImportResult
	json.NewDecoder(resp.Body).Decode(&result)
	resp.Body.Close()
	if result.Imported != 4 {
		t.Errorf("re-imported = %d, want 4", result.Imported)
	}

	// Verify each value survived the round trip.
	cases := []struct {
		key    string
		verify func(raw json.RawMessage) bool
	}{
		{"plain", func(raw json.RawMessage) bool {
			return entryValue(raw) == "hello"
		}},
		{"jsonobj", func(raw json.RawMessage) bool {
			var obj map[string]interface{}
			return json.Unmarshal(raw, &obj) == nil && obj["x"] == float64(1)
		}},
		{"arr", func(raw json.RawMessage) bool {
			var arr []string
			return json.Unmarshal(raw, &arr) == nil && len(arr) == 3
		}},
		{"empty", func(raw json.RawMessage) bool {
			return entryValue(raw) == ""
		}},
	}
	for _, tc := range cases {
		resp, _ := client.Get(ts.URL + "/api/entries/" + tc.key)
		var e apiEntry
		json.NewDecoder(resp.Body).Decode(&e)
		resp.Body.Close()
		if !tc.verify(e.Value) {
			t.Errorf("key %q: value did not survive export/import round trip: %s", tc.key, e.Value)
		}
	}
}

// TestAPIResponseFieldsNeverRemoved is a regression guard that checks all
// required response fields are present on every write and read endpoint.
// Any field removal would break existing scripts and clients.
func TestAPIResponseFieldsNeverRemoved(t *testing.T) {
	ts, client := startServer(t)

	entryFields := []string{"id", "category", "key", "value", "created_at", "updated_at"}
	listFields := []string{"entries", "total", "page", "per_page", "total_pages"}

	checkFields := func(t *testing.T, label string, body []byte, required []string) {
		t.Helper()
		var raw map[string]json.RawMessage
		json.Unmarshal(body, &raw)
		for _, field := range required {
			if _, ok := raw[field]; !ok {
				t.Errorf("%s: missing required field %q — removing fields is a breaking change", label, field)
			}
		}
	}

	readBody := func(resp *http.Response) []byte {
		var buf bytes.Buffer
		buf.ReadFrom(resp.Body)
		resp.Body.Close()
		return buf.Bytes()
	}

	// POST /api/entries
	resp, _ := client.Post(ts.URL+"/api/entries", "application/json",
		bytes.NewReader([]byte(`{"category":"compat","key":"stable","value":"v"}`)))
	body := readBody(resp)
	checkFields(t, "POST /api/entries", body, entryFields)

	// Extract ID from response for subsequent calls.
	var created struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(body, &created)

	// GET /api/entries/:id (numeric)
	resp, _ = client.Get(fmt.Sprintf("%s/api/entries/%d", ts.URL, created.ID))
	checkFields(t, "GET /api/entries/:id", readBody(resp), entryFields)

	// GET /api/entries/:key
	resp, _ = client.Get(ts.URL + "/api/entries/stable")
	checkFields(t, "GET /api/entries/:key", readBody(resp), entryFields)

	// PUT /api/entries/:key
	req, _ := http.NewRequest("PUT", ts.URL+"/api/entries/stable",
		bytes.NewReader([]byte(`{"value":"updated"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	checkFields(t, "PUT /api/entries/:key", readBody(resp), entryFields)

	// GET /api/entries (list)
	resp, _ = client.Get(ts.URL + "/api/entries")
	checkFields(t, "GET /api/entries", readBody(resp), listFields)
}

// TestNumericIDsAlwaysWork is a regression guard ensuring the original numeric ID
// lookup mechanism still works. Scripts written before key lookup existed use IDs.
func TestNumericIDsAlwaysWork(t *testing.T) {
	ts, client := startServer(t)

	e := postEntry(t, client, ts.URL, map[string]string{"category": "test", "key": "id_test", "value": "val"})
	if e.ID == 0 {
		t.Fatal("expected non-zero ID")
	}

	idURL := fmt.Sprintf("%s/api/entries/%d", ts.URL, e.ID)

	// GET
	resp, _ := client.Get(idURL)
	if resp.StatusCode != 200 {
		t.Errorf("GET by numeric ID: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// PUT
	req, _ := http.NewRequest("PUT", idURL, bytes.NewReader([]byte(`{"value":"updated"}`)))
	req.Header.Set("Content-Type", "application/json")
	resp, _ = client.Do(req)
	if resp.StatusCode != 200 {
		t.Errorf("PUT by numeric ID: status = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// DELETE
	req, _ = http.NewRequest("DELETE", idURL, nil)
	resp, _ = client.Do(req)
	if resp.StatusCode != 204 {
		t.Errorf("DELETE by numeric ID: status = %d, want 204", resp.StatusCode)
	}
	resp.Body.Close()
}
