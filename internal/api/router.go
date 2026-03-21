package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/chinmay28/homeapi/internal/middleware"
)

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

			// Try to serve the file; if not found, serve index.html for SPA routing
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}

			// Check if file exists
			f, err := frontendFS.Open(strings.TrimPrefix(path, "/"))
			if err != nil {
				// Serve index.html for SPA client-side routing
				r.URL.Path = "/index.html"
			} else {
				f.Close()
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
