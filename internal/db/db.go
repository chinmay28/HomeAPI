package db

import (
	"database/sql"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/chinmay28/homeapi/internal/models"
)

// Store provides database operations for entries.
type Store struct {
	db *sql.DB
}

// New creates a new Store with the given database path.
// If dbPath is empty, it defaults to ~/.homeapi/homeapi.db.
func New(dbPath string) (*Store, error) {
	if dbPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home dir: %w", err)
		}
		dbPath = filepath.Join(home, ".homeapi", "homeapi.db")
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// NewInMemory creates a Store backed by an in-memory SQLite database.
// Useful for testing.
func NewInMemory() (*Store, error) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("opening in-memory database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return s, nil
}

// Close closes the database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS entries (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		category   TEXT NOT NULL DEFAULT 'default',
		key        TEXT NOT NULL,
		value      TEXT NOT NULL DEFAULT '',
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(category, key)
	);
	CREATE INDEX IF NOT EXISTS idx_entries_category ON entries(category);
	CREATE INDEX IF NOT EXISTS idx_entries_key ON entries(key);
	`
	_, err := s.db.Exec(schema)
	return err
}

// CreateEntry inserts a new entry into the database.
func (s *Store) CreateEntry(e *models.Entry) (*models.Entry, error) {
	if e.Category == "" {
		e.Category = "default"
	}

	now := time.Now().UTC()
	result, err := s.db.Exec(
		"INSERT INTO entries (category, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		e.Category, e.Key, e.Value, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("inserting entry: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("getting last insert id: %w", err)
	}

	e.ID = id
	e.CreatedAt = now
	e.UpdatedAt = now
	return e, nil
}

// GetEntry retrieves an entry by ID.
func (s *Store) GetEntry(id int64) (*models.Entry, error) {
	var e models.Entry
	err := s.db.QueryRow(
		"SELECT id, category, key, value, created_at, updated_at FROM entries WHERE id = ?", id,
	).Scan(&e.ID, &e.Category, &e.Key, &e.Value, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying entry: %w", err)
	}
	return &e, nil
}

// UpdateEntry updates an existing entry.
func (s *Store) UpdateEntry(id int64, category, key, value *string) (*models.Entry, error) {
	existing, err := s.GetEntry(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, nil
	}

	if category != nil {
		existing.Category = *category
	}
	if key != nil {
		existing.Key = *key
	}
	if value != nil {
		existing.Value = *value
	}

	now := time.Now().UTC()
	_, err = s.db.Exec(
		"UPDATE entries SET category = ?, key = ?, value = ?, updated_at = ? WHERE id = ?",
		existing.Category, existing.Key, existing.Value, now, id,
	)
	if err != nil {
		return nil, fmt.Errorf("updating entry: %w", err)
	}

	existing.UpdatedAt = now
	return existing, nil
}

// DeleteEntry deletes an entry by ID. Returns true if deleted.
func (s *Store) DeleteEntry(id int64) (bool, error) {
	result, err := s.db.Exec("DELETE FROM entries WHERE id = ?", id)
	if err != nil {
		return false, fmt.Errorf("deleting entry: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("getting rows affected: %w", err)
	}
	return rows > 0, nil
}

// ListEntries returns a paginated list of entries, optionally filtered.
func (s *Store) ListEntries(params models.ListParams) (*models.PaginatedEntries, error) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PerPage < 1 || params.PerPage > 200 {
		params.PerPage = 50
	}

	where := "1=1"
	args := []interface{}{}

	if params.Category != "" {
		where += " AND category = ?"
		args = append(args, params.Category)
	}
	if params.Search != "" {
		where += " AND (key LIKE ? OR value LIKE ?)"
		searchTerm := "%" + params.Search + "%"
		args = append(args, searchTerm, searchTerm)
	}

	// Count total
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := s.db.QueryRow("SELECT COUNT(*) FROM entries WHERE "+where, countArgs...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("counting entries: %w", err)
	}

	// Fetch page
	offset := (params.Page - 1) * params.PerPage
	args = append(args, params.PerPage, offset)
	rows, err := s.db.Query(
		"SELECT id, category, key, value, created_at, updated_at FROM entries WHERE "+where+" ORDER BY updated_at DESC LIMIT ? OFFSET ?",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("querying entries: %w", err)
	}
	defer rows.Close()

	entries := []models.Entry{}
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.Category, &e.Key, &e.Value, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning entry: %w", err)
		}
		entries = append(entries, e)
	}

	totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

	return &models.PaginatedEntries{
		Entries:    entries,
		Total:      total,
		Page:       params.Page,
		PerPage:    params.PerPage,
		TotalPages: totalPages,
	}, nil
}

// ListCategories returns all categories with entry counts.
func (s *Store) ListCategories() ([]models.CategoryInfo, error) {
	rows, err := s.db.Query("SELECT category, COUNT(*) as count FROM entries GROUP BY category ORDER BY category")
	if err != nil {
		return nil, fmt.Errorf("querying categories: %w", err)
	}
	defer rows.Close()

	var categories []models.CategoryInfo
	for rows.Next() {
		var c models.CategoryInfo
		if err := rows.Scan(&c.Name, &c.Count); err != nil {
			return nil, fmt.Errorf("scanning category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, nil
}

// ExportAll returns all entries for export.
func (s *Store) ExportAll() ([]models.Entry, error) {
	rows, err := s.db.Query("SELECT id, category, key, value, created_at, updated_at FROM entries ORDER BY category, key")
	if err != nil {
		return nil, fmt.Errorf("exporting entries: %w", err)
	}
	defer rows.Close()

	var entries []models.Entry
	for rows.Next() {
		var e models.Entry
		if err := rows.Scan(&e.ID, &e.Category, &e.Key, &e.Value, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ImportEntries imports entries with merge or replace semantics.
func (s *Store) ImportEntries(entries []models.Entry, mode string) (*models.ImportResult, error) {
	result := &models.ImportResult{}

	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	for _, e := range entries {
		if e.Key == "" {
			result.Errors++
			continue
		}
		if e.Category == "" {
			e.Category = "default"
		}

		now := time.Now().UTC()

		if mode == "replace" {
			_, err := tx.Exec(
				`INSERT INTO entries (category, key, value, created_at, updated_at)
				 VALUES (?, ?, ?, ?, ?)
				 ON CONFLICT(category, key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
				e.Category, e.Key, e.Value, now, now,
			)
			if err != nil {
				result.Errors++
				continue
			}
			result.Imported++
		} else {
			// merge: skip if exists
			var exists int
			err := tx.QueryRow("SELECT COUNT(*) FROM entries WHERE category = ? AND key = ?", e.Category, e.Key).Scan(&exists)
			if err != nil {
				result.Errors++
				continue
			}
			if exists > 0 {
				result.Skipped++
				continue
			}
			_, err = tx.Exec(
				"INSERT INTO entries (category, key, value, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
				e.Category, e.Key, e.Value, now, now,
			)
			if err != nil {
				result.Errors++
				continue
			}
			result.Imported++
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("committing transaction: %w", err)
	}

	return result, nil
}
