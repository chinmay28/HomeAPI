package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

// seedEntries creates the fixture entries used by the bulk delete tests.
func seedEntries(t *testing.T, h *Handler) {
	t.Helper()
	for _, e := range []map[string]string{
		{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."},
		{"category": "watchlist", "key": "GOOGL", "value": "Google LLC"},
		{"category": "watchlist", "key": "MSFT", "value": "Microsoft Corp."},
		{"category": "config", "key": "thermostat", "value": "72"},
		{"category": "notes", "key": "groceries", "value": "milk, eggs"},
	} {
		body, _ := json.Marshal(e)
		req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
		w := httptest.NewRecorder()
		h.CreateEntry(w, req)
		if w.Code != 201 {
			t.Fatalf("seed %s: status = %d, body: %s", e["key"], w.Code, w.Body.String())
		}
	}
}

// bulkDelete issues a DELETE against /api/entries and decodes the result.
func bulkDelete(t *testing.T, h *Handler, target string, body io.Reader) (int, models.BulkDeleteResult) {
	t.Helper()
	req := httptest.NewRequest("DELETE", target, body)
	w := httptest.NewRecorder()
	h.BulkDeleteEntries(w, req)

	var result models.BulkDeleteResult
	json.Unmarshal(w.Body.Bytes(), &result)
	return w.Code, result
}

func remainingEntries(t *testing.T, h *Handler) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/api/entries?per_page=200", nil)
	w := httptest.NewRecorder()
	h.ListEntries(w, req)

	var list struct {
		Total int `json:"total"`
	}
	json.NewDecoder(w.Body).Decode(&list)
	return list.Total
}

func TestBulkDeleteEntries_QueryParams(t *testing.T) {
	tests := []struct {
		name        string
		target      string
		wantStatus  int
		wantDeleted int
		wantRemain  int
	}{
		{
			name:        "single key",
			target:      "/api/entries?key=AAPL",
			wantStatus:  200,
			wantDeleted: 1,
			wantRemain:  4,
		},
		{
			name:        "repeated key params",
			target:      "/api/entries?key=AAPL&key=GOOGL",
			wantStatus:  200,
			wantDeleted: 2,
			wantRemain:  3,
		},
		{
			name:        "comma separated keys",
			target:      "/api/entries?keys=AAPL,GOOGL,MSFT",
			wantStatus:  200,
			wantDeleted: 3,
			wantRemain:  2,
		},
		{
			name:        "by category",
			target:      "/api/entries?category=watchlist",
			wantStatus:  200,
			wantDeleted: 3,
			wantRemain:  2,
		},
		{
			name:        "by search",
			target:      "/api/entries?search=Apple",
			wantStatus:  200,
			wantDeleted: 1,
			wantRemain:  4,
		},
		{
			name:        "by id",
			target:      "/api/entries?id=1",
			wantStatus:  200,
			wantDeleted: 1,
			wantRemain:  4,
		},
		{
			name:        "comma separated ids",
			target:      "/api/entries?ids=1,2",
			wantStatus:  200,
			wantDeleted: 2,
			wantRemain:  3,
		},
		{
			name:        "category and search combined",
			target:      "/api/entries?category=watchlist&search=Google",
			wantStatus:  200,
			wantDeleted: 1,
			wantRemain:  4,
		},
		{
			name:        "no match is not an error",
			target:      "/api/entries?key=DOESNOTEXIST",
			wantStatus:  200,
			wantDeleted: 0,
			wantRemain:  5,
		},
		{
			name:        "all deletes everything",
			target:      "/api/entries?all=true",
			wantStatus:  200,
			wantDeleted: 5,
			wantRemain:  0,
		},
		{
			name:       "no selector is rejected",
			target:     "/api/entries",
			wantStatus: 400,
			wantRemain: 5,
		},
		{
			name:       "all=false without selector is rejected",
			target:     "/api/entries?all=false",
			wantStatus: 400,
			wantRemain: 5,
		},
		{
			name:       "non-numeric id is rejected",
			target:     "/api/entries?id=abc",
			wantStatus: 400,
			wantRemain: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)
			seedEntries(t, h)

			status, result := bulkDelete(t, h, tt.target, nil)
			if status != tt.wantStatus {
				t.Fatalf("status = %d, want %d", status, tt.wantStatus)
			}
			if tt.wantStatus == 200 {
				if result.Deleted != tt.wantDeleted {
					t.Errorf("deleted = %d, want %d", result.Deleted, tt.wantDeleted)
				}
				if len(result.Entries) != tt.wantDeleted {
					t.Errorf("entries = %d, want %d", len(result.Entries), tt.wantDeleted)
				}
			}
			if got := remainingEntries(t, h); got != tt.wantRemain {
				t.Errorf("remaining = %d, want %d", got, tt.wantRemain)
			}
		})
	}
}

func TestBulkDeleteEntries_JSONBody(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	body, _ := json.Marshal(map[string]interface{}{"keys": []string{"AAPL", "GOOGL"}})
	status, result := bulkDelete(t, h, "/api/entries", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if got := remainingEntries(t, h); got != 3 {
		t.Errorf("remaining = %d, want 3", got)
	}
}

func TestBulkDeleteEntries_JSONBodyWithIDsAndFilters(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	body, _ := json.Marshal(map[string]interface{}{
		"ids":      []int64{1},
		"category": "watchlist",
	})
	status, result := bulkDelete(t, h, "/api/entries", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", result.Deleted)
	}
	if len(result.Entries) != 1 || result.Entries[0].Key != "AAPL" {
		t.Errorf("entries = %+v, want AAPL only", result.Entries)
	}
}

func TestBulkDeleteEntries_BodyOverridesQuery(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	// category in the body wins over the query parameter; keys are additive.
	body, _ := json.Marshal(map[string]interface{}{"category": "config"})
	status, result := bulkDelete(t, h, "/api/entries?category=watchlist", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if result.Deleted != 1 || result.Entries[0].Key != "thermostat" {
		t.Errorf("deleted = %d entries = %+v, want just thermostat", result.Deleted, result.Entries)
	}
}

func TestBulkDeleteEntries_DryRun(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	status, result := bulkDelete(t, h, "/api/entries?category=watchlist&dry_run=true", nil)
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if !result.DryRun {
		t.Error("dry_run = false, want true")
	}
	if result.Matched != 3 {
		t.Errorf("matched = %d, want 3", result.Matched)
	}
	if result.Deleted != 0 {
		t.Errorf("deleted = %d, want 0", result.Deleted)
	}
	if got := remainingEntries(t, h); got != 5 {
		t.Errorf("remaining = %d, want 5 (dry run must not delete)", got)
	}
}

func TestBulkDeleteEntries_DryRunViaBody(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	body, _ := json.Marshal(map[string]interface{}{"keys": []string{"AAPL"}, "dry_run": true})
	status, result := bulkDelete(t, h, "/api/entries", bytes.NewReader(body))
	if status != 200 {
		t.Fatalf("status = %d, want 200", status)
	}
	if !result.DryRun || result.Deleted != 0 || result.Matched != 1 {
		t.Errorf("result = %+v, want dry run with 1 match and 0 deleted", result)
	}
	if got := remainingEntries(t, h); got != 5 {
		t.Errorf("remaining = %d, want 5", got)
	}
}

func TestBulkDeleteEntries_InvalidJSONBody(t *testing.T) {
	h := newTestHandler(t)
	seedEntries(t, h)

	status, _ := bulkDelete(t, h, "/api/entries?key=AAPL", bytes.NewReader([]byte("{not json")))
	if status != 400 {
		t.Fatalf("status = %d, want 400", status)
	}
	if got := remainingEntries(t, h); got != 5 {
		t.Errorf("remaining = %d, want 5 (nothing deleted on bad body)", got)
	}
}

func TestBulkDeleteEntries_ErrorResponseShape(t *testing.T) {
	h := newTestHandler(t)

	req := httptest.NewRequest("DELETE", "/api/entries", nil)
	w := httptest.NewRecorder()
	h.BulkDeleteEntries(w, req)

	var resp models.ErrorResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Code != "VALIDATION_ERROR" {
		t.Errorf("code = %q, want VALIDATION_ERROR", resp.Code)
	}
	if resp.Error == "" {
		t.Error("expected a non-empty error message")
	}
}

func TestSplitList(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "empty", in: nil, want: nil},
		{name: "single", in: []string{"a"}, want: []string{"a"}},
		{name: "repeated", in: []string{"a", "b"}, want: []string{"a", "b"}},
		{name: "comma separated", in: []string{"a,b"}, want: []string{"a", "b"}},
		{name: "mixed with spaces", in: []string{"a, b", "c"}, want: []string{"a", "b", "c"}},
		{name: "empty items dropped", in: []string{"a,,b", ""}, want: []string{"a", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitList(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestQueryBool(t *testing.T) {
	tests := []struct {
		target string
		want   bool
	}{
		{target: "/api/entries", want: false},
		{target: "/api/entries?all", want: true},
		{target: "/api/entries?all=", want: true},
		{target: "/api/entries?all=1", want: true},
		{target: "/api/entries?all=true", want: true},
		{target: "/api/entries?all=TRUE", want: true},
		{target: "/api/entries?all=yes", want: true},
		{target: "/api/entries?all=0", want: false},
		{target: "/api/entries?all=false", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			req := httptest.NewRequest("DELETE", tt.target, nil)
			if got := queryBool(req, "all"); got != tt.want {
				t.Errorf("queryBool(%q) = %v, want %v", tt.target, got, tt.want)
			}
		})
	}
}
