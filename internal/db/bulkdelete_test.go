package db

import (
	"errors"
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

// seedBulk inserts a fixed set of entries used by the bulk delete tests.
func seedBulk(t *testing.T, s *Store) {
	t.Helper()
	entries := []models.Entry{
		{Category: "watchlist", Key: "AAPL", Value: "Apple Inc."},
		{Category: "watchlist", Key: "GOOGL", Value: "Google LLC"},
		{Category: "watchlist", Key: "MSFT", Value: "Microsoft Corp."},
		{Category: "config", Key: "thermostat", Value: "72"},
		{Category: "config", Key: "apple_tv", Value: "living room"},
		{Category: "notes", Key: "groceries", Value: "milk, eggs"},
	}
	for i := range entries {
		if _, err := s.CreateEntry(&entries[i]); err != nil {
			t.Fatalf("CreateEntry(%s): %v", entries[i].Key, err)
		}
	}
}

func countEntries(t *testing.T, s *Store) int {
	t.Helper()
	result, err := s.ListEntries(models.ListParams{Page: 1, PerPage: 200})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	return result.Total
}

func TestBulkDeleteEntries(t *testing.T) {
	tests := []struct {
		name        string
		params      models.BulkDeleteParams
		wantDeleted int
		wantKeys    []string
	}{
		{
			name:        "by single key",
			params:      models.BulkDeleteParams{Keys: []string{"AAPL"}},
			wantDeleted: 1,
			wantKeys:    []string{"AAPL"},
		},
		{
			name:        "by multiple keys",
			params:      models.BulkDeleteParams{Keys: []string{"AAPL", "GOOGL"}},
			wantDeleted: 2,
			wantKeys:    []string{"AAPL", "GOOGL"},
		},
		{
			name:        "unknown key deletes nothing",
			params:      models.BulkDeleteParams{Keys: []string{"NOPE"}},
			wantDeleted: 0,
			wantKeys:    []string{},
		},
		{
			name:        "by category",
			params:      models.BulkDeleteParams{Category: "watchlist"},
			wantDeleted: 3,
			wantKeys:    []string{"AAPL", "GOOGL", "MSFT"},
		},
		{
			name:        "by search matches key and value",
			params:      models.BulkDeleteParams{Search: "apple"},
			wantDeleted: 2,
			wantKeys:    []string{"AAPL", "apple_tv"},
		},
		{
			name:        "search narrowed by category",
			params:      models.BulkDeleteParams{Category: "config", Search: "apple"},
			wantDeleted: 1,
			wantKeys:    []string{"apple_tv"},
		},
		{
			name:        "keys narrowed by category",
			params:      models.BulkDeleteParams{Category: "config", Keys: []string{"AAPL", "thermostat"}},
			wantDeleted: 1,
			wantKeys:    []string{"thermostat"},
		},
		{
			name:        "keys and ids are combined",
			params:      models.BulkDeleteParams{Keys: []string{"AAPL"}, IDs: []int64{4}},
			wantDeleted: 2,
			wantKeys:    []string{"AAPL", "thermostat"},
		},
		{
			name:        "all deletes everything",
			params:      models.BulkDeleteParams{All: true},
			wantDeleted: 6,
			wantKeys:    []string{"AAPL", "GOOGL", "MSFT", "thermostat", "apple_tv", "groceries"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestStore(t)
			seedBulk(t, s)

			result, err := s.BulkDeleteEntries(tt.params)
			if err != nil {
				t.Fatalf("BulkDeleteEntries: %v", err)
			}
			if result.Deleted != tt.wantDeleted {
				t.Errorf("deleted = %d, want %d", result.Deleted, tt.wantDeleted)
			}
			if result.Matched != tt.wantDeleted {
				t.Errorf("matched = %d, want %d", result.Matched, tt.wantDeleted)
			}
			if result.DryRun {
				t.Error("dry_run = true, want false")
			}
			if len(result.Entries) != len(tt.wantKeys) {
				t.Fatalf("entries = %d, want %d (%v)", len(result.Entries), len(tt.wantKeys), result.Entries)
			}
			got := map[string]bool{}
			for _, e := range result.Entries {
				got[e.Key] = true
				if e.ID == 0 || e.Category == "" {
					t.Errorf("deleted entry %+v missing id or category", e)
				}
			}
			for _, k := range tt.wantKeys {
				if !got[k] {
					t.Errorf("expected %q among deleted keys, got %v", k, result.Entries)
				}
			}

			// Remaining rows must be exactly the ones not matched.
			if remaining := countEntries(t, s); remaining != 6-tt.wantDeleted {
				t.Errorf("remaining = %d, want %d", remaining, 6-tt.wantDeleted)
			}
		})
	}
}

func TestBulkDeleteEntries_DryRun(t *testing.T) {
	s := newTestStore(t)
	seedBulk(t, s)

	result, err := s.BulkDeleteEntries(models.BulkDeleteParams{Category: "watchlist", DryRun: true})
	if err != nil {
		t.Fatalf("BulkDeleteEntries: %v", err)
	}
	if !result.DryRun {
		t.Error("dry_run = false, want true")
	}
	if result.Matched != 3 {
		t.Errorf("matched = %d, want 3", result.Matched)
	}
	if result.Deleted != 0 {
		t.Errorf("deleted = %d, want 0", result.Deleted)
	}
	if len(result.Entries) != 3 {
		t.Errorf("entries = %d, want 3", len(result.Entries))
	}
	if remaining := countEntries(t, s); remaining != 6 {
		t.Errorf("remaining = %d, want 6 (dry run must not delete)", remaining)
	}
}

func TestBulkDeleteEntries_NoSelector(t *testing.T) {
	s := newTestStore(t)
	seedBulk(t, s)

	_, err := s.BulkDeleteEntries(models.BulkDeleteParams{})
	if !errors.Is(err, ErrNoBulkDeleteFilter) {
		t.Fatalf("err = %v, want ErrNoBulkDeleteFilter", err)
	}
	if remaining := countEntries(t, s); remaining != 6 {
		t.Errorf("remaining = %d, want 6 (nothing should be deleted)", remaining)
	}
}

func TestBulkDeleteEntries_DuplicateKeysDeleteOnce(t *testing.T) {
	s := newTestStore(t)
	seedBulk(t, s)

	result, err := s.BulkDeleteEntries(models.BulkDeleteParams{Keys: []string{"AAPL", "AAPL"}, IDs: []int64{1}})
	if err != nil {
		t.Fatalf("BulkDeleteEntries: %v", err)
	}
	if result.Deleted != 1 {
		t.Errorf("deleted = %d, want 1", result.Deleted)
	}
	if remaining := countEntries(t, s); remaining != 5 {
		t.Errorf("remaining = %d, want 5", remaining)
	}
}

func TestBulkDeleteEntries_SameKeyAcrossCategories(t *testing.T) {
	s := newTestStore(t)
	// The unique constraint is (category, key), so the same key can live in
	// two categories. A key selector removes both unless a category narrows it.
	for _, e := range []models.Entry{
		{Category: "a", Key: "shared", Value: "1"},
		{Category: "b", Key: "shared", Value: "2"},
	} {
		entry := e
		if _, err := s.CreateEntry(&entry); err != nil {
			t.Fatalf("CreateEntry: %v", err)
		}
	}

	result, err := s.BulkDeleteEntries(models.BulkDeleteParams{Keys: []string{"shared"}})
	if err != nil {
		t.Fatalf("BulkDeleteEntries: %v", err)
	}
	if result.Deleted != 2 {
		t.Errorf("deleted = %d, want 2", result.Deleted)
	}
	if remaining := countEntries(t, s); remaining != 0 {
		t.Errorf("remaining = %d, want 0", remaining)
	}
}
