package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chinmay28/homeapi/internal/db"
	"github.com/chinmay28/homeapi/internal/models"
)

const Version = "1.0.0"

// Handler holds the API handlers and their dependencies.
type Handler struct {
	store *db.Store
}

// NewHandler creates a new Handler with the given store.
func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

// Health returns service health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, models.HealthResponse{
		Status:  "ok",
		Version: Version,
	})
}

// ListEntries returns a paginated list of entries.
func (h *Handler) ListEntries(w http.ResponseWriter, r *http.Request) {
	params := models.ListParams{
		Category: r.URL.Query().Get("category"),
		Search:   r.URL.Query().Get("search"),
		Page:     queryInt(r, "page", 1),
		PerPage:  queryInt(r, "per_page", 50),
	}

	result, err := h.store.ListEntries(params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list entries", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateEntry creates a new entry.
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	var entry models.Entry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "VALIDATION_ERROR")
		return
	}

	if msg := entry.Validate(); msg != "" {
		writeError(w, http.StatusBadRequest, msg, "VALIDATION_ERROR")
		return
	}

	created, err := h.store.CreateEntry(&entry)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, fmt.Sprintf("Entry with category %q and key %q already exists", entry.Category, entry.Key), "CONFLICT")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to create entry", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// GetEntry returns a single entry by ID.
func (h *Handler) GetEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	entry, err := h.store.GetEntry(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get entry", "INTERNAL_ERROR")
		return
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "Entry not found", "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, entry)
}

// UpdateEntry updates an existing entry.
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "VALIDATION_ERROR")
		return
	}

	var category, key, value *string
	if v, ok := body["category"].(string); ok {
		category = &v
	}
	if v, ok := body["key"].(string); ok {
		key = &v
	}
	if v, ok := body["value"].(string); ok {
		value = &v
	}

	updated, err := h.store.UpdateEntry(id, category, key, value)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			writeError(w, http.StatusConflict, "An entry with that category and key already exists", "CONFLICT")
			return
		}
		writeError(w, http.StatusInternalServerError, "Failed to update entry", "INTERNAL_ERROR")
		return
	}
	if updated == nil {
		writeError(w, http.StatusNotFound, "Entry not found", "NOT_FOUND")
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// DeleteEntry deletes an entry by ID.
func (h *Handler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	deleted, err := h.store.DeleteEntry(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to delete entry", "INTERNAL_ERROR")
		return
	}
	if !deleted {
		writeError(w, http.StatusNotFound, "Entry not found", "NOT_FOUND")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListCategories returns all categories with counts.
func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.store.ListCategories()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to list categories", "INTERNAL_ERROR")
		return
	}
	if categories == nil {
		categories = []models.CategoryInfo{}
	}
	writeJSON(w, http.StatusOK, categories)
}

// ExportData exports all entries as JSON.
func (h *Handler) ExportData(w http.ResponseWriter, r *http.Request) {
	entries, err := h.store.ExportAll()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to export data", "INTERNAL_ERROR")
		return
	}
	if entries == nil {
		entries = []models.Entry{}
	}

	export := models.ExportData{
		Version:    "1",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Entries:    entries,
	}

	filename := fmt.Sprintf("homeapi-export-%s.json", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	writeJSON(w, http.StatusOK, export)
}

// ImportData imports entries from JSON.
func (h *Handler) ImportData(w http.ResponseWriter, r *http.Request) {
	var req models.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "VALIDATION_ERROR")
		return
	}

	if len(req.Entries) == 0 {
		writeError(w, http.StatusBadRequest, "No entries to import", "VALIDATION_ERROR")
		return
	}

	if req.Mode == "" {
		req.Mode = "merge"
	}
	if req.Mode != "merge" && req.Mode != "replace" {
		writeError(w, http.StatusBadRequest, "Mode must be 'merge' or 'replace'", "VALIDATION_ERROR")
		return
	}

	result, err := h.store.ImportEntries(req.Entries, req.Mode)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to import data", "INTERNAL_ERROR")
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// Helper functions

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message, code string) {
	writeJSON(w, status, models.ErrorResponse{Error: message, Code: code})
}

func queryInt(r *http.Request, key string, defaultVal int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(val)
	if err != nil || n < 1 {
		return defaultVal
	}
	return n
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	// Extract ID from URL path: /api/entries/{id}
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		writeError(w, http.StatusBadRequest, "Invalid entry ID", "VALIDATION_ERROR")
		return 0, false
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid entry ID", "VALIDATION_ERROR")
		return 0, false
	}
	return id, true
}
