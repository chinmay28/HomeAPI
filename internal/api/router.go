package api

import (
	"io/fs"
	"net/http"
	pathpkg "path"
	"strings"

	"github.com/chinmay28/homeapi/internal/middleware"
)

// serveAppShell writes the SPA's index.html, which every client-side route
// resolves to. It is deliberately not cached: the bundles it points at carry
// content hashes in their names, so the shell is the one file that must always
// come back fresh after an upgrade.
func serveAppShell(w http.ResponseWriter, frontendFS fs.FS) {
	index, err := fs.ReadFile(frontendFS, "index.html")
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	w.Write(index)
}

// NewRouter creates the HTTP router with all API routes and static file serving.
func NewRouter(h *Handler, frontendFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/health", methodHandler(h.Health, "GET"))
	mux.HandleFunc("/api/entries", func(w http.ResponseWriter, r *http.Request) {
		// /api/entries exactly (no trailing path)
		if r.URL.Path != "/api/entries" && r.URL.Path != "/api/entries/" {
			// This is /api/entries/{id}
			switch r.Method {
			case http.MethodGet:
				h.GetEntry(w, r)
			case http.MethodPut:
				h.UpdateEntry(w, r)
			case http.MethodDelete:
				h.DeleteEntry(w, r)
			default:
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.ListEntries(w, r)
		case http.MethodPost:
			h.CreateEntry(w, r)
		case http.MethodDelete:
			// Bulk delete by key, ID, category or search query.
			h.BulkDeleteEntries(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	// Catch /api/entries/{id} paths
	mux.HandleFunc("/api/entries/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetEntry(w, r)
		case http.MethodPut:
			h.UpdateEntry(w, r)
		case http.MethodDelete:
			h.DeleteEntry(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/categories", methodHandler(h.ListCategories, "GET"))
	mux.HandleFunc("/api/export", methodHandler(h.ExportData, "GET"))
	mux.HandleFunc("/api/import", methodHandler(h.ImportData, "POST"))

	// Serve frontend static files
	if frontendFS != nil {
		fileServer := http.FileServer(http.FS(frontendFS))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// For API paths that didn't match, return 404
			if strings.HasPrefix(r.URL.Path, "/api/") {
				writeError(w, http.StatusNotFound, "API endpoint not found", "NOT_FOUND")
				return
			}

			// Real files (the hashed JS/CSS bundles, icons, the badge images)
			// go to the file server; everything else is a client-side route and
			// gets the app shell so deep links and refreshes work.
			//
			// The shell is written out here rather than by rewriting the path
			// and delegating: http.FileServer redirects any request ending in
			// "index.html" to "./", which the browser resolves against the URL
			// it actually asked for — so "/entries/9" bounced to "/entries/",
			// which bounced to itself, and deep links died in a redirect loop.
			name := strings.TrimPrefix(pathpkg.Clean("/"+r.URL.Path), "/")
			if name == "" {
				serveAppShell(w, frontendFS)
				return
			}
			f, err := frontendFS.Open(name)
			if err != nil {
				serveAppShell(w, frontendFS)
				return
			}
			info, statErr := f.Stat()
			f.Close()
			// Directories have no index of their own to show — a URL that
			// happens to name one (e.g. /static) is still a client route.
			if statErr != nil || info.IsDir() {
				serveAppShell(w, frontendFS)
				return
			}
			fileServer.ServeHTTP(w, r)
		})
	}

	// Apply middleware
	var handler http.Handler = mux
	handler = middleware.CORS(handler)
	handler = middleware.Logger(handler)

	return handler
}

func methodHandler(h http.HandlerFunc, methods ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, m := range methods {
			if r.Method == m {
				h(w, r)
				return
			}
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
