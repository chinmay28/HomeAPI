package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

// TestDisplay_Compat: a database with no minion entries serves a frame, and
// nothing about the existing endpoints changes because the display ones exist.
func TestDisplay_Compat(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	// A frame from an empty store.
	resp, err := http.Get(ts.URL + "/api/displays/minion3/frame.bin")
	if err != nil {
		t.Fatalf("GET frame.bin: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("frame.bin status = %d, want 200", resp.StatusCode)
	}
	var body bytes.Buffer
	body.ReadFrom(resp.Body)
	if body.Len() != 4000 {
		t.Errorf("frame.bin body = %d bytes, want 4000", body.Len())
	}
	for _, h := range []string{"ETag", "X-Minion-Format", "X-Minion-Sleep-Seconds", "X-Minion-Next-Wake", "X-Minion-Rendered-At"} {
		if resp.Header.Get(h) == "" {
			t.Errorf("frame.bin missing header %s", h)
		}
	}
	if resp.Header.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("frame.bin Content-Type = %s", resp.Header.Get("Content-Type"))
	}

	// The PNG preview too.
	pngResp, err := http.Get(ts.URL + "/api/displays/minion3/frame.png?scale=1")
	if err != nil {
		t.Fatalf("GET frame.png: %v", err)
	}
	pngResp.Body.Close()
	if pngResp.StatusCode != 200 || pngResp.Header.Get("Content-Type") != "image/png" {
		t.Errorf("frame.png status %d type %s", pngResp.StatusCode, pngResp.Header.Get("Content-Type"))
	}

	// Serving frames from an empty store wrote nothing: the entries list and
	// the category list are exactly what a pre-display server would return.
	listResp, err := http.Get(ts.URL + "/api/entries")
	if err != nil {
		t.Fatal(err)
	}
	defer listResp.Body.Close()
	var list struct {
		Entries    []json.RawMessage `json:"entries"`
		Total      int               `json:"total"`
		Page       int               `json:"page"`
		PerPage    int               `json:"per_page"`
		TotalPages int               `json:"total_pages"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode entries: %v", err)
	}
	if list.Total != 0 || len(list.Entries) != 0 || list.Page != 1 || list.PerPage != 50 {
		t.Errorf("entries list changed: %+v", list)
	}
	catResp, err := http.Get(ts.URL + "/api/categories")
	if err != nil {
		t.Fatal(err)
	}
	var cats []json.RawMessage
	json.NewDecoder(catResp.Body).Decode(&cats)
	catResp.Body.Close()
	if len(cats) != 0 {
		t.Errorf("categories = %d, want none", len(cats))
	}

	// Existing entries keep their response shape with the display mounted.
	createBody, _ := json.Marshal(map[string]string{"category": "watchlist", "key": "AAPL", "value": "Apple Inc."})
	createResp, err := http.Post(ts.URL+"/api/entries", "application/json", bytes.NewReader(createBody))
	if err != nil {
		t.Fatal(err)
	}
	createResp.Body.Close()
	if createResp.StatusCode != 201 {
		t.Fatalf("create status = %d", createResp.StatusCode)
	}
	getResp, err := http.Get(ts.URL + "/api/entries/AAPL")
	if err != nil {
		t.Fatal(err)
	}
	defer getResp.Body.Close()
	var got map[string]json.RawMessage
	json.NewDecoder(getResp.Body).Decode(&got)
	for _, field := range []string{"id", "category", "key", "value", "value_text", "created_at", "updated_at"} {
		if _, ok := got[field]; !ok {
			t.Errorf("entry response lost field %q", field)
		}
	}
	if entryValue(got["value"]) != "Apple Inc." {
		t.Errorf("entry value = %s", got["value"])
	}

	// Unknown displays are ordinary JSON 404s, like any unknown API path.
	nf, err := http.Get(ts.URL + "/api/displays/other/frame.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer nf.Body.Close()
	var apiErr map[string]string
	json.NewDecoder(nf.Body).Decode(&apiErr)
	if nf.StatusCode != 404 || apiErr["code"] != "NOT_FOUND" {
		t.Errorf("unknown display: status %d body %v", nf.StatusCode, apiErr)
	}
}
