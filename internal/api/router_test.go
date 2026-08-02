package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
)

// A stand-in for the embedded CRA build: an app shell plus a hashed asset.
func testFrontend() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                 {Data: []byte("<!doctype html><div id=root></div>")},
		"icon.svg":                   {Data: []byte("<svg/>")},
		"static/js/main.abc123.js":   {Data: []byte("console.log(1)")},
		"static/css/main.abc123.css": {Data: []byte(".app{}")},
	}
}

func TestServesRealFiles(t *testing.T) {
	router := NewRouter(newTestHandler(t), testFrontend())

	for _, path := range []string{"/icon.svg", "/static/js/main.abc123.js"} {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, w.Code)
		}
		if w.Body.Len() == 0 {
			t.Errorf("GET %s: empty body", path)
		}
	}
}

// Every client-side route has to answer with the app shell directly. Rewriting
// the path and handing it to http.FileServer used to bounce "/entries/9" to
// "/entries/", which bounced to itself — deep links and page refreshes died in
// a redirect loop.
func TestClientRoutesServeAppShellWithoutRedirecting(t *testing.T) {
	router := NewRouter(newTestHandler(t), testFrontend())

	paths := []string{"/", "/entries", "/entries/9", "/entries/some-key", "/settings", "/static"}
	for _, path := range paths {
		req := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200 (Location: %q)", path, w.Code, w.Header().Get("Location"))
			continue
		}
		if got := w.Body.String(); got != "<!doctype html><div id=root></div>" {
			t.Errorf("GET %s: body = %q, want the app shell", path, got)
		}
		if got, want := w.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
			t.Errorf("GET %s: Content-Type = %q, want %q", path, got, want)
		}
	}
}

// The shell points at content-hashed bundles, so it is the one file that must
// never be served from a stale cache after an upgrade.
func TestAppShellIsNotCached(t *testing.T) {
	router := NewRouter(newTestHandler(t), testFrontend())

	req := httptest.NewRequest("GET", "/entries/9", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if got, want := w.Header().Get("Cache-Control"), "no-cache"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

// Static serving must never swallow an unknown API path — scripts rely on a
// JSON 404 there, not on the HTML app shell.
func TestUnknownAPIPathStays404JSON(t *testing.T) {
	router := NewRouter(newTestHandler(t), testFrontend())

	req := httptest.NewRequest("GET", "/api/nope", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
	if got, want := w.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

// Existing curl scripts address entries by numeric ID; the static handler sits
// on "/" and must not shadow them.
func TestAPIRoutesStillResolveWithFrontendMounted(t *testing.T) {
	h := newTestHandler(t)
	router := NewRouter(h, testFrontend())

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got, want := w.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

// A build without an embedded frontend (`go build` straight from source) still
// has to serve the API.
func TestNilFrontendLeavesAPIWorking(t *testing.T) {
	router := NewRouter(newTestHandler(t), nil)

	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}
