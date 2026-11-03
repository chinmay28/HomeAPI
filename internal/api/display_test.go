package api

import (
	"bytes"
	"encoding/json"
	"image/png"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/chinmay28/homeapi/internal/display"
	"github.com/chinmay28/homeapi/internal/models"
)

const (
	testQuotes  = `{"BTC-USD":"109432.12","IBIT":"61.30","STRC":"101.5","VTI":"301.20","GLD":"214.05","P":"58.41","ORCL":"190.77","timestamp":"11/03 07:10:00"}`
	testQuotes2 = `{"BTC-USD":"110000","IBIT":"62.00","STRC":"101.5","VTI":"302.00","GLD":"214.00","P":"58.00","ORCL":"191.00","timestamp":"11/03 19:10:00"}`
	testAt      = "2026-11-03T07:15:00-08:00"
)

// seed creates an entry directly in the handler's store.
func seed(t *testing.T, h *Handler, category, key, value string) {
	t.Helper()
	if _, err := h.store.CreateEntry(&models.Entry{Category: category, Key: key, Value: value}); err != nil {
		t.Fatalf("seed %s: %v", key, err)
	}
}

// newDisplayHandler is a handler with the minion entries a real deployment has.
func newDisplayHandler(t *testing.T) *Handler {
	t.Helper()
	h := newTestHandler(t)
	cfg, _ := json.Marshal(display.DefaultConfig())
	seed(t, h, "minion", display.ParamsKey, string(cfg))
	seed(t, h, "minion", display.QuotesKey, testQuotes)
	seed(t, h, "minion", "minion-sum", "42")
	return h
}

func getFrame(h *Handler, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.Display(w, req)
	return w
}

func entryValueByKey(t *testing.T, h *Handler, key string) (string, bool) {
	t.Helper()
	e, err := h.store.GetEntryByKey(key)
	if err != nil {
		t.Fatalf("GetEntryByKey(%s): %v", key, err)
	}
	if e == nil {
		return "", false
	}
	return e.Value, true
}

var etagRE = regexp.MustCompile(`^"[0-9a-f]{16}"$`)

func TestFrameHandlers(t *testing.T) {
	h := newDisplayHandler(t)

	t.Run("frame.bin headers and body", func(t *testing.T) {
		w := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt+"&wake=183", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
		}
		if w.Body.Len() != display.NativeSize {
			t.Errorf("body = %d bytes, want %d", w.Body.Len(), display.NativeSize)
		}
		want := map[string]string{
			"Content-Type":           "application/octet-stream",
			"Content-Length":         "4000",
			"Cache-Control":          "no-cache",
			"X-Minion-Format":        "1",
			"X-Minion-Sleep-Seconds": "43200",
			"X-Minion-Next-Wake":     "2026-11-03T19:15:00-08:00",
			"X-Minion-Rendered-At":   testAt,
		}
		for k, v := range want {
			if got := w.Header().Get(k); got != v {
				t.Errorf("%s = %q, want %q", k, got, v)
			}
		}
		if got := w.Header().Get("ETag"); !etagRE.MatchString(got) {
			t.Errorf("ETag = %q, want a quoted 16-hex strong tag", got)
		}
		if w.Header().Get("X-Minion-Stale") != "" {
			t.Error("fresh quotes must not be marked stale")
		}

		// The body is exactly what the renderer produces for these inputs.
		loc, _ := time.LoadLocation("America/Los_Angeles")
		now := time.Date(2026, 11, 3, 7, 15, 0, 0, loc)
		frame, err := display.Render(display.DefaultConfig(), display.Inputs{
			Quotes: display.ParseQuotes(testQuotes, now, loc), Sum: "42", Wake: 183, HasWake: true,
		}, now)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(w.Body.Bytes(), frame.Native()) {
			t.Error("body differs from display.Render for the same inputs")
		}
	})

	t.Run("304 on matching ETag keeps the headers", func(t *testing.T) {
		first := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil)
		etag := first.Header().Get("ETag")
		for _, inm := range []string{etag, "W/" + etag, `"other", ` + etag, "*"} {
			w := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, map[string]string{"If-None-Match": inm})
			if w.Code != http.StatusNotModified {
				t.Errorf("If-None-Match %q: status = %d, want 304", inm, w.Code)
			}
			if w.Body.Len() != 0 {
				t.Errorf("If-None-Match %q: 304 must carry no body", inm)
			}
			for _, k := range []string{"ETag", "X-Minion-Sleep-Seconds", "X-Minion-Next-Wake", "X-Minion-Rendered-At", "X-Minion-Format"} {
				if w.Header().Get(k) == "" {
					t.Errorf("If-None-Match %q: %s missing on 304", inm, k)
				}
			}
		}
		w := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, map[string]string{"If-None-Match": `"0000000000000000"`})
		if w.Code != http.StatusOK {
			t.Errorf("stale ETag: status = %d, want 200", w.Code)
		}
	})

	t.Run("ETag follows the picture", func(t *testing.T) {
		a := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil).Header().Get("ETag")
		b := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt+"&wake=9", nil).Header().Get("ETag")
		c := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil).Header().Get("ETag")
		if a != c {
			t.Error("same inputs should give the same ETag")
		}
		if a == b {
			t.Error("a different wake counter changes the footer and so the ETag")
		}
	})

	t.Run("at is honoured", func(t *testing.T) {
		w := getFrame(h, "/api/displays/minion3/frame.bin?at=2026-11-03T19:15:00-08:00", nil)
		if got := w.Header().Get("X-Minion-Sleep-Seconds"); got != "43200" {
			t.Errorf("sleep = %s, want 43200", got)
		}
		if got := w.Header().Get("X-Minion-Next-Wake"); got != "2026-11-04T07:15:00-08:00" {
			t.Errorf("next wake = %s", got)
		}
		// Four hours after the 07:10 quotes the picture is stale.
		w = getFrame(h, "/api/displays/minion3/frame.bin?at=2026-11-03T11:15:00-08:00", nil)
		if w.Header().Get("X-Minion-Stale") != "1" {
			t.Error("old quotes should set X-Minion-Stale: 1")
		}
		// A UTC instant is converted to the schedule's zone.
		w = getFrame(h, "/api/displays/minion3/frame.bin?at=2026-11-03T15:15:00Z", nil)
		if got := w.Header().Get("X-Minion-Rendered-At"); got != testAt {
			t.Errorf("rendered-at = %s, want %s", got, testAt)
		}
	})

	t.Run("frame.png", func(t *testing.T) {
		for scale, want := range map[string]int{"": 3, "1": 1, "2": 2, "8": 8} {
			path := "/api/displays/minion3/frame.png?at=" + testAt
			if scale != "" {
				path += "&scale=" + scale
			}
			w := getFrame(h, path, nil)
			if w.Code != http.StatusOK {
				t.Fatalf("scale %q: status = %d", scale, w.Code)
			}
			if ct := w.Header().Get("Content-Type"); ct != "image/png" {
				t.Errorf("scale %q: Content-Type = %s", scale, ct)
			}
			for _, k := range []string{"X-Minion-Sleep-Seconds", "X-Minion-Next-Wake", "X-Minion-Rendered-At", "X-Minion-Format", "ETag"} {
				if w.Header().Get(k) == "" {
					t.Errorf("scale %q: %s missing on png", scale, k)
				}
			}
			img, err := png.Decode(w.Body)
			if err != nil {
				t.Fatalf("scale %q: decode: %v", scale, err)
			}
			if b := img.Bounds(); b.Dx() != display.Width*want || b.Dy() != display.Height*want {
				t.Errorf("scale %q: %dx%d, want %dx%d", scale, b.Dx(), b.Dy(), display.Width*want, display.Height*want)
			}
		}
		bin := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil).Header().Get("ETag")
		p3 := getFrame(h, "/api/displays/minion3/frame.png?at="+testAt, nil).Header().Get("ETag")
		p2 := getFrame(h, "/api/displays/minion3/frame.png?at="+testAt+"&scale=2", nil).Header().Get("ETag")
		if bin == p3 || p3 == p2 {
			t.Error("different bodies must not share an ETag")
		}
	})

	t.Run("bad query parameters", func(t *testing.T) {
		for _, path := range []string{
			"/api/displays/minion3/frame.bin?at=yesterday",
			"/api/displays/minion3/frame.bin?wake=-1",
			"/api/displays/minion3/frame.bin?wake=abc",
			"/api/displays/minion3/frame.png?scale=0",
			"/api/displays/minion3/frame.png?scale=9",
			"/api/displays/minion3/frame.png?scale=big",
		} {
			w := getFrame(h, path, nil)
			if w.Code != http.StatusBadRequest {
				t.Errorf("%s: status = %d, want 400", path, w.Code)
			}
			var e models.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&e); err != nil || e.Code != "VALIDATION_ERROR" {
				t.Errorf("%s: body not a VALIDATION_ERROR: %v %+v", path, err, e)
			}
		}
	})

	t.Run("unknown display or file is a JSON 404", func(t *testing.T) {
		for _, path := range []string{
			"/api/displays/minion4/frame.bin",
			"/api/displays/minion3/frame.gif",
			"/api/displays/minion3",
			"/api/displays/minion3/",
			"/api/displays/",
		} {
			w := getFrame(h, path, nil)
			if w.Code != http.StatusNotFound {
				t.Errorf("%s: status = %d, want 404", path, w.Code)
			}
			var e models.ErrorResponse
			if err := json.NewDecoder(w.Body).Decode(&e); err != nil || e.Code != "NOT_FOUND" {
				t.Errorf("%s: body not a NOT_FOUND error", path)
			}
		}
	})

	t.Run("through the router", func(t *testing.T) {
		router := NewRouter(h, testFrontend())
		req := httptest.NewRequest("GET", "/api/displays/minion3/frame.bin?at="+testAt, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK || w.Body.Len() != display.NativeSize {
			t.Errorf("router GET: status %d, %d bytes", w.Code, w.Body.Len())
		}
		req = httptest.NewRequest("POST", "/api/displays/minion3/frame.bin", strings.NewReader(""))
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusMethodNotAllowed {
			t.Errorf("router POST: status %d, want 405", w.Code)
		}
	})
}

func TestFrameHandlers_Degraded(t *testing.T) {
	t.Run("empty store still serves a frame", func(t *testing.T) {
		h := newTestHandler(t)
		w := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil)
		if w.Code != http.StatusOK || w.Body.Len() != display.NativeSize {
			t.Fatalf("status %d, %d bytes", w.Code, w.Body.Len())
		}
		if w.Header().Get("X-Minion-Sleep-Seconds") != "43200" {
			t.Errorf("default schedule should still apply: %s", w.Header().Get("X-Minion-Sleep-Seconds"))
		}
		if w.Header().Get("X-Minion-Stale") != "1" {
			t.Error("no quotes at all should be marked stale")
		}
	})

	t.Run("invalid params fall back to the default and still render", func(t *testing.T) {
		h := newTestHandler(t)
		seed(t, h, "minion", display.ParamsKey, "{not json")
		seed(t, h, "minion", display.QuotesKey, testQuotes)
		w := getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt, nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d", w.Code)
		}
		good := newDisplayHandler(t)
		ok := getFrame(good, "/api/displays/minion3/frame.bin?at="+testAt, nil)
		if bytes.Equal(w.Body.Bytes(), ok.Body.Bytes()) {
			t.Error("the fallback frame should differ (it shows Cfg?)")
		}
	})

	t.Run("custom params change the picture and the schedule", func(t *testing.T) {
		h := newTestHandler(t)
		seed(t, h, "minion", display.ParamsKey, `{"hero":{"key":"GLD","label":"GOLD","prefix":"$","decimals":2},"schedule":{"tz":"UTC","wake_times":["00:00"]},"sum":{"key":"note"}}`)
		seed(t, h, "minion", display.QuotesKey, testQuotes)
		seed(t, h, "notes", "note", "hello world")
		w := getFrame(h, "/api/displays/minion3/frame.bin?at=2026-11-03T12:00:00Z", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d", w.Code)
		}
		if got := w.Header().Get("X-Minion-Next-Wake"); got != "2026-11-04T00:00:00Z" {
			t.Errorf("next wake = %s", got)
		}
		if got := w.Header().Get("X-Minion-Sleep-Seconds"); got != "43200" {
			t.Errorf("sleep = %s", got)
		}
	})
}

func TestFrameSideEffects(t *testing.T) {
	t.Run("device wake records last-seen and the quotes snapshot", func(t *testing.T) {
		h := newDisplayHandler(t)
		before := time.Now().UTC().Truncate(time.Second)
		w := getFrame(h, "/api/displays/minion3/frame.bin?wake=5", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("status %d", w.Code)
		}
		seen, ok := entryValueByKey(t, h, display.LastSeenKey)
		if !ok {
			t.Fatal("minion3-last-seen not written")
		}
		var rec struct {
			Wake int    `json:"wake"`
			At   string `json:"at"`
		}
		if err := json.Unmarshal([]byte(seen), &rec); err != nil || rec.Wake != 5 {
			t.Errorf("last-seen = %s", seen)
		}
		at, err := time.Parse(time.RFC3339, rec.At)
		if err != nil || at.Before(before) {
			t.Errorf("last-seen at = %q, want a current RFC 3339 time", rec.At)
		}
		e, _ := h.store.GetEntryByKey(display.LastSeenKey)
		if e.Category != display.Category {
			t.Errorf("last-seen category = %q, want %q", e.Category, display.Category)
		}

		prev, ok := entryValueByKey(t, h, display.PrevQuotesKey)
		if !ok || prev != testQuotes {
			t.Errorf("prev-quotes = %q, want the current quotes text", prev)
		}
	})

	t.Run("snapshot is replaced only when the quotes timestamp changes", func(t *testing.T) {
		h := newDisplayHandler(t)
		// A snapshot with the same timestamp but different prices stands in
		// for "already captured this fetch".
		sentinel := `{"VTI":"1.00","timestamp":"11/03 07:10:00"}`
		seed(t, h, display.Category, display.PrevQuotesKey, sentinel)
		getFrame(h, "/api/displays/minion3/frame.bin", nil)
		if prev, _ := entryValueByKey(t, h, display.PrevQuotesKey); prev != sentinel {
			t.Errorf("snapshot replaced although the timestamp did not change: %s", prev)
		}
		// New fetch → new timestamp → the snapshot moves on.
		q, _ := h.store.GetEntryByKey(display.QuotesKey)
		v := testQuotes2
		if _, err := h.store.UpdateEntry(q.ID, nil, nil, &v); err != nil {
			t.Fatal(err)
		}
		getFrame(h, "/api/displays/minion3/frame.bin", nil)
		if prev, _ := entryValueByKey(t, h, display.PrevQuotesKey); prev != testQuotes2 {
			t.Errorf("snapshot not replaced after the timestamp changed: %s", prev)
		}
	})

	t.Run("304 still records the wake", func(t *testing.T) {
		h := newDisplayHandler(t)
		first := getFrame(h, "/api/displays/minion3/frame.bin?wake=1", nil)
		w := getFrame(h, "/api/displays/minion3/frame.bin?wake=1", map[string]string{"If-None-Match": first.Header().Get("ETag")})
		if w.Code != http.StatusNotModified {
			// The clock in the footer has minute resolution; a tick between
			// the two requests legitimately changes the frame.
			t.Skip("frame changed between requests")
		}
		q, _ := h.store.GetEntryByKey(display.QuotesKey)
		v := testQuotes2
		h.store.UpdateEntry(q.ID, nil, nil, &v)
		w = getFrame(h, "/api/displays/minion3/frame.bin?wake=2", map[string]string{"If-None-Match": first.Header().Get("ETag")})
		if w.Code == http.StatusNotModified {
			if prev, _ := entryValueByKey(t, h, display.PrevQuotesKey); prev != testQuotes2 {
				t.Error("snapshot should move on even on a 304")
			}
		}
		seen, _ := entryValueByKey(t, h, display.LastSeenKey)
		if !strings.Contains(seen, `"wake":2`) {
			t.Errorf("last-seen = %s, want wake 2", seen)
		}
	})

	t.Run("previews never write", func(t *testing.T) {
		h := newDisplayHandler(t)
		getFrame(h, "/api/displays/minion3/frame.png?wake=7", nil)
		getFrame(h, "/api/displays/minion3/frame.png?scale=2", nil)
		getFrame(h, "/api/displays/minion3/frame.bin?at="+testAt+"&wake=7", nil)
		if _, ok := entryValueByKey(t, h, display.LastSeenKey); ok {
			t.Error("a preview wrote minion3-last-seen")
		}
		if _, ok := entryValueByKey(t, h, display.PrevQuotesKey); ok {
			t.Error("a preview wrote minion3-prev-quotes")
		}
	})

	t.Run("no wake parameter leaves last-seen alone", func(t *testing.T) {
		h := newDisplayHandler(t)
		getFrame(h, "/api/displays/minion3/frame.bin", nil)
		if _, ok := entryValueByKey(t, h, display.LastSeenKey); ok {
			t.Error("last-seen written without a wake counter")
		}
		if _, ok := entryValueByKey(t, h, display.PrevQuotesKey); !ok {
			t.Error("the quotes snapshot should still be taken")
		}
	})

	t.Run("no quotes means no snapshot", func(t *testing.T) {
		h := newTestHandler(t)
		getFrame(h, "/api/displays/minion3/frame.bin?wake=1", nil)
		if _, ok := entryValueByKey(t, h, display.PrevQuotesKey); ok {
			t.Error("an empty snapshot was written")
		}
	})
}

func TestEtagMatches(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"", false},
		{`"abc"`, true},
		{`W/"abc"`, true},
		{`"x", "abc"`, true},
		{`"x",W/"abc"`, true},
		{"*", true},
		{`"abd"`, false},
		{`abc`, false},
	}
	for _, tt := range tests {
		if got := etagMatches(tt.header, `"abc"`); got != tt.want {
			t.Errorf("etagMatches(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}
