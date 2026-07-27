package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

// bulkDeleteRequest sends DELETE /api/entries and decodes the result.
func bulkDeleteRequest(t *testing.T, client *http.Client, url, query string, body io.Reader) (int, models.BulkDeleteResult) {
	t.Helper()
	req, err := http.NewRequest("DELETE", url+"/api/entries"+query, body)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("DELETE /api/entries%s: %v", query, err)
	}
	defer resp.Body.Close()

	var result models.BulkDeleteResult
	json.NewDecoder(resp.Body).Decode(&result)
	return resp.StatusCode, result
}

func listAll(t *testing.T, client *http.Client, url string) apiList {
	t.Helper()
	resp, err := client.Get(url + "/api/entries?per_page=200")
	if err != nil {
		t.Fatalf("GET /api/entries: %v", err)
	}
	defer resp.Body.Close()

	var list apiList
	json.NewDecoder(resp.Body).Decode(&list)
	return list
}

// TestBulkDeleteWorkflow walks a realistic cleanup: preview with a dry run,
// delete a couple of keys, prune a whole category, then clear what is left.
func TestBulkDeleteWorkflow(t *testing.T) {
	ts, client := startServer(t)

	for _, e := range []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."},
		{"category": "watchlist", "key": "GOOGL", "value": "Alphabet Inc."},
		{"category": "watchlist", "key": "MSFT", "value": "Microsoft Corp."},
		{"category": "config", "key": "thermostat_temp", "value": "72"},
		{"category": "config", "key": "alarm_time", "value": "07:00"},
		{"category": "notes", "key": "groceries", "value": "milk, eggs"},
	} {
		postEntry(t, client, ts.URL, e)
	}

	// Step 1: preview a category delete without touching anything.
	status, result := bulkDeleteRequest(t, client, ts.URL, "?category=watchlist&dry_run=true", nil)
	if status != 200 {
		t.Fatalf("dry run status = %d, want 200", status)
	}
	if !result.DryRun {
		t.Error("dry_run = false, want true")
	}
	if result.Matched != 3 || result.Deleted != 0 {
		t.Errorf("matched = %d deleted = %d, want 3 and 0", result.Matched, result.Deleted)
	}
	if got := listAll(t, client, ts.URL).Total; got != 6 {
		t.Fatalf("after dry run total = %d, want 6", got)
	}

	// Step 2: delete two specific keys.
	status, result = bulkDeleteRequest(t, client, ts.URL, "?key=AAPL&key=GOOGL", nil)
	if status != 200 {
		t.Fatalf("key delete status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if got := listAll(t, client, ts.URL).Total; got != 4 {
		t.Fatalf("after key delete total = %d, want 4", got)
	}

	// Step 3: prune the config category with a JSON body.
	body, _ := json.Marshal(map[string]interface{}{"category": "config"})
	status, result = bulkDeleteRequest(t, client, ts.URL, "", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("category delete status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}

	// Step 4: only MSFT and groceries survive.
	list := listAll(t, client, ts.URL)
	if list.Total != 2 {
		t.Fatalf("total = %d, want 2", list.Total)
	}
	surviving := map[string]bool{}
	for _, e := range list.Entries {
		surviving[e.Key] = true
	}
	for _, k := range []string{"MSFT", "groceries"} {
		if !surviving[k] {
			t.Errorf("expected %q to survive, got %v", k, surviving)
		}
	}

	// Step 5: clear everything that is left.
	status, result = bulkDeleteRequest(t, client, ts.URL, "?all=true", nil)
	if status != 200 {
		t.Fatalf("all delete status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if got := listAll(t, client, ts.URL).Total; got != 0 {
		t.Errorf("final total = %d, want 0", got)
	}

	// Step 6: categories are empty too.
	resp, _ := client.Get(ts.URL + "/api/categories")
	var categories []models.CategoryInfo
	json.NewDecoder(resp.Body).Decode(&categories)
	resp.Body.Close()
	if len(categories) != 0 {
		t.Errorf("categories = %v, want none", categories)
	}
}

// TestBulkDeleteNeverWipesWithoutSelector guards the safety rule: an
// unfiltered bulk delete must be rejected rather than emptying the database.
func TestBulkDeleteNeverWipesWithoutSelector(t *testing.T) {
	ts, client := startServer(t)
	postEntry(t, client, ts.URL, map[string]string{"category": "config", "key": "keepme", "value": "important"})

	for _, query := range []string{"", "?all=false", "?dry_run=true"} {
		status, _ := bulkDeleteRequest(t, client, ts.URL, query, nil)
		if status != 400 {
			t.Errorf("DELETE /api/entries%s status = %d, want 400", query, status)
		}
	}

	if got := listAll(t, client, ts.URL).Total; got != 1 {
		t.Errorf("total = %d, want 1", got)
	}
}
