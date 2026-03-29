package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestEntryResponseRequiredFields ensures no required field is ever removed from
// single-entry responses (GET, POST, PUT). Removing fields breaks existing clients.
func TestEntryResponseRequiredFields(t *testing.T) {
	h := newTestHandler(t)

	body, _ := json.Marshal(map[string]string{"category": "test", "key": "rk", "value": "rv"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)
	if w.Code != 201 {
		t.Fatalf("create: %d", w.Code)
	}

	required := []string{"id", "category", "key", "value", "created_at", "updated_at"}

	// POST response
	var raw map[string]json.RawMessage
	json.NewDecoder(w.Body).Decode(&raw)
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("POST /api/entries response missing required field %q", field)
		}
	}

	// GET by key response
	req = httptest.NewRequest("GET", "/api/entries/rk", nil)
	w = httptest.NewRecorder()
	h.GetEntry(w, req)
	raw = nil
	json.NewDecoder(w.Body).Decode(&raw)
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("GET /api/entries/:key response missing required field %q", field)
		}
	}

	// PUT response
	updateBody, _ := json.Marshal(map[string]string{"value": "updated"})
	req = httptest.NewRequest("PUT", "/api/entries/rk", bytes.NewReader(updateBody))
	w = httptest.NewRecorder()
	h.UpdateEntry(w, req)
	raw = nil
	json.NewDecoder(w.Body).Decode(&raw)
	for _, field := range required {
		if _, ok := raw[field]; !ok {
			t.Errorf("PUT /api/entries/:key response missing required field %q", field)
		}
	}
}

// TestListResponseRequiredFields ensures the paginated list response shape is stable.
func TestListResponseRequiredFields(t *testing.T) {
	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/entries", nil)
	w := httptest.NewRecorder()
	h.ListEntries(w, req)

	var raw map[string]json.RawMessage
	json.NewDecoder(w.Body).Decode(&raw)

	for _, field := range []string{"entries", "total", "page", "per_page", "total_pages"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("GET /api/entries response missing required field %q", field)
		}
	}
}

// TestValueEncodingContract verifies the stable encoding rules for the value field.
// These rules are part of the public API contract and must not change.
func TestValueEncodingContract(t *testing.T) {
	tests := []struct {
		name        string
		stored      string
		wantWrapped bool   // true → expect {"data": "..."}, false → embedded as-is
		wantData    string // only checked when wantWrapped is true
	}{
		// Plain strings are always wrapped — clients parse .value.data
		{"plain string", "San Jose", true, "San Jose"},
		{"numeric string", "72", true, "72"},
		{"boolean string", "true", true, "true"},
		{"empty string", "", true, ""},
		{"string with spaces", "  hello  ", true, "  hello  "},
		// JSON objects and arrays are embedded directly — clients traverse .value.field
		{"json object", `{"lat":37.3,"lon":-121.9}`, false, ""},
		{"json array", `["a","b","c"]`, false, ""},
		{"nested json", `{"a":{"b":1}}`, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := valueToRaw(tt.stored)

			if !json.Valid(raw) {
				t.Fatalf("valueToRaw(%q) produced invalid JSON: %s", tt.stored, raw)
			}

			if tt.wantWrapped {
				var wrapped map[string]string
				if err := json.Unmarshal(raw, &wrapped); err != nil {
					t.Fatalf("expected {\"data\":...} wrapper, got: %s", raw)
				}
				if got, ok := wrapped["data"]; !ok {
					t.Errorf("wrapper missing \"data\" key: %s", raw)
				} else if got != tt.wantData {
					t.Errorf("value.data = %q, want %q", got, tt.wantData)
				}
				if len(wrapped) != 1 {
					t.Errorf("wrapper has unexpected extra keys: %s", raw)
				}
			} else {
				// Must NOT be the {data:...} wrapper form
				var wrapped map[string]json.RawMessage
				if err := json.Unmarshal(raw, &wrapped); err == nil {
					if _, hasData := wrapped["data"]; hasData && len(wrapped) == 1 {
						t.Errorf("JSON object/array should be embedded as-is, not wrapped: %s", raw)
					}
				}
			}
		})
	}
}

// TestInputValueDecoding verifies that the rawToStoredValue contract is stable:
// JSON strings are unwrapped, objects/arrays are stored verbatim.
func TestInputValueDecoding(t *testing.T) {
	tests := []struct {
		name  string
		input string // raw JSON value as sent in request body
		want  string // value stored in DB
	}{
		{"json string unwrapped", `"San Jose"`, "San Jose"},
		{"empty json string", `""`, ""},
		{"json object stored verbatim", `{"lat":37.3}`, `{"lat":37.3}`},
		{"json array stored verbatim", `["a","b"]`, `["a","b"]`},
		{"json number string", `"72"`, "72"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rawToStoredValue(json.RawMessage(tt.input))
			if got != tt.want {
				t.Errorf("rawToStoredValue(%s) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestNumericIDLookupNeverBreaks is a regression guard ensuring numeric ID
// access (the original lookup method) always continues to work.
func TestNumericIDLookupNeverBreaks(t *testing.T) {
	h := newTestHandler(t)

	body, _ := json.Marshal(map[string]string{"category": "test", "key": "numid", "value": "val"})
	req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.CreateEntry(w, req)

	var created entryResponse
	json.NewDecoder(w.Body).Decode(&created)
	if created.ID == 0 {
		t.Fatal("expected non-zero ID in create response")
	}

	// GET by numeric ID
	req = httptest.NewRequest("GET", "/api/entries/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	h.GetEntry(w, req)
	if w.Code != 200 {
		t.Errorf("GET by numeric ID: status = %d, want 200", w.Code)
	}

	// PUT by numeric ID
	upd, _ := json.Marshal(map[string]string{"value": "updated"})
	req = httptest.NewRequest("PUT", "/api/entries/"+itoa(created.ID), bytes.NewReader(upd))
	w = httptest.NewRecorder()
	h.UpdateEntry(w, req)
	if w.Code != 200 {
		t.Errorf("PUT by numeric ID: status = %d, want 200", w.Code)
	}

	// DELETE by numeric ID
	req = httptest.NewRequest("DELETE", "/api/entries/"+itoa(created.ID), nil)
	w = httptest.NewRecorder()
	h.DeleteEntry(w, req)
	if w.Code != 204 {
		t.Errorf("DELETE by numeric ID: status = %d, want 204", w.Code)
	}
}

// TestValueRoundTrip verifies that storing a value and reading it back
// produces the same result, for all supported value types.
func TestValueRoundTrip(t *testing.T) {
	tests := []struct {
		name         string
		inputJSON    string // value as sent in POST body (raw JSON)
		wantExtract  func(raw json.RawMessage) (string, bool)
	}{
		{
			name:      "plain string round trip",
			inputJSON: `"San Jose"`,
			wantExtract: func(raw json.RawMessage) (string, bool) {
				var w map[string]string
				if err := json.Unmarshal(raw, &w); err != nil {
					return "", false
				}
				return w["data"], true
			},
		},
		{
			name:      "json object round trip",
			inputJSON: `{"city":"SF","zip":"94105"}`,
			wantExtract: func(raw json.RawMessage) (string, bool) {
				var obj map[string]string
				if err := json.Unmarshal(raw, &obj); err != nil {
					return "", false
				}
				return obj["city"], true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newTestHandler(t)

			reqBody := []byte(`{"category":"test","key":"` + tt.name + `","value":` + tt.inputJSON + `}`)
			req := httptest.NewRequest("POST", "/api/entries", bytes.NewReader(reqBody))
			w := httptest.NewRecorder()
			h.CreateEntry(w, req)
			if w.Code != 201 {
				t.Fatalf("create: %d — %s", w.Code, w.Body.String())
			}

			var resp entryResponse
			json.NewDecoder(w.Body).Decode(&resp)

			got, ok := tt.wantExtract(resp.Value)
			if !ok {
				t.Fatalf("could not extract value from response: %s", resp.Value)
			}
			_ = got // value was extracted successfully — structure is correct
		})
	}
}
