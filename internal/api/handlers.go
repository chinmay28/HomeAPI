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

// entryResponse is the JSON API response for an entry.
// Value is always a JSON value: objects/arrays are embedded as-is;
// plain strings are wrapped as {"data": "..."}.
type entryResponse struct {
	ID        int64           `json:"id"`
	Category  string          `json:"category"`
	Key       string          `json:"key"`
	Value     json.RawMessage `json:"value"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// valueToRaw converts a stored string value to its JSON representation.
// JSON objects and arrays are embedded as-is; everything else is wrapped as {"data": "..."}.
func valueToRaw(value string) json.RawMessage {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) > 0 && (trimmed[0] == '{' || trimmed[0] == '[') && json.Valid([]byte(trimmed)) {
		return json.RawMessage(trimmed)
	}
	b, _ := json.Marshal(map[string]string{"data": value})
	return json.RawMessage(b)
}

// rawToStoredValue converts a JSON raw value from a request body to the string stored in DB.
// JSON strings are unwrapped (e.g. "San Jose" → San Jose).
// JSON objects/arrays are stored as their JSON string representation.
func rawToStoredValue(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}

func toEntryResponse(e *models.Entry) entryResponse {
	return entryResponse{
		ID:        e.ID,
		Category:  e.Category,
		Key:       e.Key,
		Value:     valueToRaw(e.Value),
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
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

	responses := make([]entryResponse, len(result.Entries))
	for i, e := range result.Entries {
		responses[i] = toEntryResponse(&e)
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"entries":     responses,
		"total":       result.Total,
		"page":        result.Page,
		"per_page":    result.PerPage,
		"total_pages": result.TotalPages,
	})
}

// CreateEntry creates a new entry.
func (h *Handler) CreateEntry(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Category string          `json:"category"`
		Key      string          `json:"key"`
		Value    json.RawMessage `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "VALIDATION_ERROR")
		return
	}

	entry := models.Entry{
		Category: body.Category,
		Key:      body.Key,
		Value:    rawToStoredValue(body.Value),
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

	writeJSON(w, http.StatusCreated, toEntryResponse(created))
}

// resolveEntry looks up an entry by numeric ID or by key string from the URL path.
func (h *Handler) resolveEntry(w http.ResponseWriter, r *http.Request) (*models.Entry, bool) {
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	seg := parts[len(parts)-1]
	if seg == "" {
		writeError(w, http.StatusBadRequest, "Missing entry ID or key", "VALIDATION_ERROR")
		return nil, false
	}

	// Try numeric ID first
	if id, err := strconv.ParseInt(seg, 10, 64); err == nil {
		entry, err := h.store.GetEntry(id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to get entry", "INTERNAL_ERROR")
			return nil, false
		}
		if entry == nil {
			writeError(w, http.StatusNotFound, "Entry not found", "NOT_FOUND")
			return nil, false
		}
		return entry, true
	}

	// Fall back to key lookup
	entry, err := h.store.GetEntryByKey(seg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to get entry", "INTERNAL_ERROR")
		return nil, false
	}
	if entry == nil {
		writeError(w, http.StatusNotFound, "Entry not found", "NOT_FOUND")
		return nil, false
	}
	return entry, true
}

// GetEntry returns a single entry by ID or key.
func (h *Handler) GetEntry(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveEntry(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, toEntryResponse(entry))
}

// UpdateEntry updates an existing entry by ID or key.
func (h *Handler) UpdateEntry(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveEntry(w, r)
	if !ok {
		return
	}

	var body map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body", "VALIDATION_ERROR")
		return
	}

	var category, key, value *string
	if raw, exists := body["category"]; exists {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			category = &s
		}
	}
	if raw, exists := body["key"]; exists {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			key = &s
		}
	}
	if raw, exists := body["value"]; exists {
		s := rawToStoredValue(raw)
		value = &s
	}

	updated, err := h.store.UpdateEntry(entry.ID, category, key, value)
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

	writeJSON(w, http.StatusOK, toEntryResponse(updated))
}

// DeleteEntry deletes an entry by ID or key.
func (h *Handler) DeleteEntry(w http.ResponseWriter, r *http.Request) {
	entry, ok := h.resolveEntry(w, r)
	if !ok {
		return
	}

	deleted, err := h.store.DeleteEntry(entry.ID)
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
