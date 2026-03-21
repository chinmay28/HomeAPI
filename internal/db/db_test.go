package db

import (
	"testing"

	"github.com/chinmay28/homeapi/internal/models"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewInMemory()
	if err != nil {
		t.Fatalf("NewInMemory: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateAndGetEntry(t *testing.T) {
	s := newTestStore(t)

	entry := &models.Entry{
		Category: "watchlist",
		Key:      "AAPL",
		Value:    "Apple Inc.",
	}

	created, err := s.CreateEntry(entry)
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if created.Category != "watchlist" {
		t.Errorf("category = %q, want %q", created.Category, "watchlist")
	}
	if created.CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt")
	}

	got, err := s.GetEntry(created.ID)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got == nil {
		t.Fatal("expected entry, got nil")
	}
	if got.Key != "AAPL" {
		t.Errorf("key = %q, want %q", got.Key, "AAPL")
	}
	if got.Value != "Apple Inc." {
		t.Errorf("value = %q, want %q", got.Value, "Apple Inc.")
	}
}

func TestCreateEntry_DefaultCategory(t *testing.T) {
	s := newTestStore(t)

	entry := &models.Entry{Key: "test", Value: "val"}
	created, err := s.CreateEntry(entry)
	if err != nil {
		t.Fatalf("CreateEntry: %v", err)
	}
	if created.Category != "default" {
		t.Errorf("category = %q, want %q", created.Category, "default")
	}
}

func TestCreateEntry_DuplicateKeyConflict(t *testing.T) {
	s := newTestStore(t)

	entry := &models.Entry{Category: "test", Key: "key1", Value: "v1"}
	_, err := s.CreateEntry(entry)
	if err != nil {
		t.Fatalf("first CreateEntry: %v", err)
	}

	dup := &models.Entry{Category: "test", Key: "key1", Value: "v2"}
	_, err = s.CreateEntry(dup)
	if err == nil {
		t.Fatal("expected error for duplicate key, got nil")
	}
}

func TestUpdateEntry(t *testing.T) {
	s := newTestStore(t)

	entry := &models.Entry{Category: "config", Key: "temp", Value: "72"}
	created, _ := s.CreateEntry(entry)

	newVal := "75"
	updated, err := s.UpdateEntry(created.ID, nil, nil, &newVal)
	if err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}
	if updated.Value != "75" {
		t.Errorf("value = %q, want %q", updated.Value, "75")
	}
	if updated.Category != "config" {
		t.Errorf("category should be unchanged, got %q", updated.Category)
	}
	if !updated.UpdatedAt.After(created.CreatedAt) || updated.UpdatedAt.Equal(created.CreatedAt) {
		// UpdatedAt should be >= CreatedAt
	}
}

func TestUpdateEntry_NotFound(t *testing.T) {
	s := newTestStore(t)

	val := "test"
	updated, err := s.UpdateEntry(999, nil, nil, &val)
	if err != nil {
		t.Fatalf("UpdateEntry: %v", err)
	}
	if updated != nil {
		t.Fatal("expected nil for non-existent entry")
	}
}

func TestDeleteEntry(t *testing.T) {
	s := newTestStore(t)

	entry := &models.Entry{Category: "test", Key: "del", Value: "v"}
	created, _ := s.CreateEntry(entry)

	deleted, err := s.DeleteEntry(created.ID)
	if err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if !deleted {
		t.Error("expected true for deleted entry")
	}

	got, _ := s.GetEntry(created.ID)
	if got != nil {
		t.Error("expected nil after deletion")
	}
}

func TestDeleteEntry_NotFound(t *testing.T) {
	s := newTestStore(t)

	deleted, err := s.DeleteEntry(999)
	if err != nil {
		t.Fatalf("DeleteEntry: %v", err)
	}
	if deleted {
		t.Error("expected false for non-existent entry")
	}
}

func TestListEntries(t *testing.T) {
	s := newTestStore(t)

	for i := 0; i < 5; i++ {
		s.CreateEntry(&models.Entry{
			Category: "cat1",
			Key:      "key" + string(rune('A'+i)),
			Value:    "val",
		})
	}
	for i := 0; i < 3; i++ {
		s.CreateEntry(&models.Entry{
			Category: "cat2",
			Key:      "key" + string(rune('X'+i)),
			Value:    "val",
		})
	}

	// All entries
	result, err := s.ListEntries(models.ListParams{Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("ListEntries: %v", err)
	}
	if result.Total != 8 {
		t.Errorf("total = %d, want 8", result.Total)
	}

	// Filter by category
	result, err = s.ListEntries(models.ListParams{Category: "cat1", Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("ListEntries (filtered): %v", err)
	}
	if result.Total != 5 {
		t.Errorf("total = %d, want 5", result.Total)
	}

	// Search
	result, err = s.ListEntries(models.ListParams{Search: "keyA", Page: 1, PerPage: 50})
	if err != nil {
		t.Fatalf("ListEntries (search): %v", err)
	}
	if result.Total != 1 {
		t.Errorf("total = %d, want 1", result.Total)
	}

	// Pagination
	result, err = s.ListEntries(models.ListParams{Page: 1, PerPage: 3})
	if err != nil {
		t.Fatalf("ListEntries (paginated): %v", err)
	}
	if len(result.Entries) != 3 {
		t.Errorf("entries = %d, want 3", len(result.Entries))
	}
	if result.TotalPages != 3 {
		t.Errorf("total_pages = %d, want 3", result.TotalPages)
	}
}

func TestListCategories(t *testing.T) {
	s := newTestStore(t)

	s.CreateEntry(&models.Entry{Category: "alpha", Key: "k1", Value: "v"})
	s.CreateEntry(&models.Entry{Category: "alpha", Key: "k2", Value: "v"})
	s.CreateEntry(&models.Entry{Category: "beta", Key: "k1", Value: "v"})

	cats, err := s.ListCategories()
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 2 {
		t.Fatalf("got %d categories, want 2", len(cats))
	}
	if cats[0].Name != "alpha" || cats[0].Count != 2 {
		t.Errorf("cats[0] = %+v, want alpha/2", cats[0])
	}
	if cats[1].Name != "beta" || cats[1].Count != 1 {
		t.Errorf("cats[1] = %+v, want beta/1", cats[1])
	}
}

func TestExportAll(t *testing.T) {
	s := newTestStore(t)

	s.CreateEntry(&models.Entry{Category: "a", Key: "k1", Value: "v1"})
	s.CreateEntry(&models.Entry{Category: "b", Key: "k2", Value: "v2"})

	entries, err := s.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestImportEntries_Merge(t *testing.T) {
	s := newTestStore(t)

	s.CreateEntry(&models.Entry{Category: "test", Key: "existing", Value: "old"})

	entries := []models.Entry{
		{Category: "test", Key: "existing", Value: "new"},
		{Category: "test", Key: "new_key", Value: "new_val"},
	}

	result, err := s.ImportEntries(entries, "merge")
	if err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}
	if result.Skipped != 1 {
		t.Errorf("skipped = %d, want 1", result.Skipped)
	}

	// Verify existing entry wasn't modified
	all, _ := s.ExportAll()
	for _, e := range all {
		if e.Key == "existing" && e.Value != "old" {
			t.Errorf("existing entry value = %q, want %q", e.Value, "old")
		}
	}
}

func TestImportEntries_Replace(t *testing.T) {
	s := newTestStore(t)

	s.CreateEntry(&models.Entry{Category: "test", Key: "existing", Value: "old"})

	entries := []models.Entry{
		{Category: "test", Key: "existing", Value: "new"},
		{Category: "test", Key: "new_key", Value: "new_val"},
	}

	result, err := s.ImportEntries(entries, "replace")
	if err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if result.Imported != 2 {
		t.Errorf("imported = %d, want 2", result.Imported)
	}

	// Verify existing entry was updated
	all, _ := s.ExportAll()
	for _, e := range all {
		if e.Key == "existing" && e.Value != "new" {
			t.Errorf("existing entry value = %q, want %q", e.Value, "new")
		}
	}
}

func TestImportEntries_InvalidKey(t *testing.T) {
	s := newTestStore(t)

	entries := []models.Entry{
		{Category: "test", Key: "", Value: "no key"},
		{Category: "test", Key: "valid", Value: "ok"},
	}

	result, err := s.ImportEntries(entries, "merge")
	if err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if result.Errors != 1 {
		t.Errorf("errors = %d, want 1", result.Errors)
	}
	if result.Imported != 1 {
		t.Errorf("imported = %d, want 1", result.Imported)
	}
}

func TestGetEntry_NotFound(t *testing.T) {
	s := newTestStore(t)

	got, err := s.GetEntry(999)
	if err != nil {
		t.Fatalf("GetEntry: %v", err)
	}
	if got != nil {
		t.Error("expected nil for non-existent entry")
	}
}
