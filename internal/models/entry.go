package models

import "time"

// Entry represents a key-value data entry in a category.
type Entry struct {
	ID        int64     `json:"id"`
	Category  string    `json:"category"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CategoryInfo holds a category name and its entry count.
type CategoryInfo struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// ExportData is the format for import/export operations.
type ExportData struct {
	Version    string  `json:"version"`
	ExportedAt string  `json:"exported_at"`
	Entries    []Entry `json:"entries"`
}

// ImportRequest is the request body for importing data.
type ImportRequest struct {
	Version    string  `json:"version,omitempty"`
	ExportedAt string  `json:"exported_at,omitempty"`
	Entries    []Entry `json:"entries"`
	Mode       string  `json:"mode"` // "merge" or "replace"
}

// ImportResult is the response for import operations.
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
	Errors   int `json:"errors"`
}

// ListParams holds query parameters for listing entries.
type ListParams struct {
	Category string
	Search   string
	Page     int
	PerPage  int
}

// BulkDeleteParams holds the selectors for a bulk delete operation.
// Keys and IDs are OR'd together; Category and Search further narrow the match.
// All must be set explicitly to delete every entry (no other selector given).
type BulkDeleteParams struct {
	Keys     []string
	IDs      []int64
	Category string
	Search   string
	All      bool
	DryRun   bool
}

// HasSelector reports whether at least one selector was provided, so that a
// bulk delete cannot wipe the database by accident.
func (p BulkDeleteParams) HasSelector() bool {
	return len(p.Keys) > 0 || len(p.IDs) > 0 || p.Category != "" || p.Search != ""
}

// DeletedEntry identifies an entry matched by a bulk delete.
type DeletedEntry struct {
	ID       int64  `json:"id"`
	Category string `json:"category"`
	Key      string `json:"key"`
}

// BulkDeleteResult is the response for bulk delete operations.
// On a dry run, Matched is populated but Deleted is 0.
type BulkDeleteResult struct {
	Deleted int            `json:"deleted"`
	Matched int            `json:"matched"`
	DryRun  bool           `json:"dry_run"`
	Entries []DeletedEntry `json:"entries"`
}

// PaginatedEntries is the response for listing entries.
type PaginatedEntries struct {
	Entries    []Entry `json:"entries"`
	Total      int     `json:"total"`
	Page       int     `json:"page"`
	PerPage    int     `json:"per_page"`
	TotalPages int     `json:"total_pages"`
}

// ErrorResponse is the standard error response format.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// HealthResponse is the response for the health check endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Validate checks that the entry has required fields.
func (e *Entry) Validate() string {
	if e.Key == "" {
		return "key is required"
	}
	if len(e.Key) > 500 {
		return "key must be 500 characters or less"
	}
	if len(e.Category) > 200 {
		return "category must be 200 characters or less"
	}
	if len(e.Value) > 100000 {
		return "value must be 100000 characters or less"
	}
	return ""
}
