package db

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chinmay28/homeapi/internal/models"
)

// testdata/mattn-go-sqlite3.db was written by HomeAPI built on
// github.com/mattn/go-sqlite3 (cgo), the driver it used before moving to the
// pure-Go modernc.org/sqlite; testdata/mattn-go-sqlite3.json is what that build
// returned for it. Rows 1, 2 and 4 went through CreateEntry, UpdateEntry and
// ImportEntries; row 3 was inserted with the column defaults, as a hand-run
// sqlite3 INSERT would be. Never regenerate these: they are the old driver's
// word on what an existing deployment's database looks like.
type mattnFixture struct {
	Raw []struct {
		ID      int64
		Created string
		Updated string
	} `json:"raw"`
	Export []models.Entry           `json:"export"`
	List   *models.PaginatedEntries `json:"list"`
}

func openMattnFixture(t *testing.T) (*Store, mattnFixture) {
	t.Helper()
	var want mattnFixture
	b, err := os.ReadFile("testdata/mattn-go-sqlite3.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &want); err != nil {
		t.Fatal(err)
	}
	// Work on a copy: opening in WAL mode writes beside the file.
	path := filepath.Join(t.TempDir(), "homeapi.db")
	src, err := os.Open("testdata/mattn-go-sqlite3.db")
	if err != nil {
		t.Fatal(err)
	}
	defer src.Close()
	dst, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(dst, src); err != nil {
		t.Fatal(err)
	}
	dst.Close()
	s, err := New(path)
	if err != nil {
		t.Fatalf("opening a database the old driver wrote: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s, want
}

func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestOldDriverDatabaseReadsIdentically: every entry of a database the cgo
// driver wrote comes back over the API byte-for-byte as it did before,
// timestamps included (UTC, "Z" suffix, nanoseconds kept).
func TestOldDriverDatabaseReadsIdentically(t *testing.T) {
	s, want := openMattnFixture(t)

	all, err := s.ExportAll()
	if err != nil {
		t.Fatal(err)
	}
	if got, w := jsonOf(t, all), jsonOf(t, want.Export); got != w {
		t.Errorf("export differs from the old driver's\n got %s\nwant %s", got, w)
	}
	list, err := s.ListEntries(models.ListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if got, w := jsonOf(t, list), jsonOf(t, want.List); got != w {
		t.Errorf("list differs from the old driver's\n got %s\nwant %s", got, w)
	}
	for _, e := range want.Export {
		got, err := s.GetEntryByKey(e.Key)
		if err != nil || got == nil {
			t.Fatalf("GetEntryByKey(%q) = %v, %v", e.Key, got, err)
		}
		if got.CreatedAt.Location() != time.UTC || got.UpdatedAt.Location() != time.UTC {
			t.Errorf("%s: times not in UTC: %v %v", e.Key, got.CreatedAt, got.UpdatedAt)
		}
	}
}

// TestNewWritesUseTheOldDriversTextFormat: rows written now are stored as the
// same text the old driver wrote, so a database mixing old and new rows still
// sorts by updated_at correctly and stays readable by an older build.
func TestNewWritesUseTheOldDriversTextFormat(t *testing.T) {
	s, want := openMattnFixture(t)

	// The fixture's own rows are untouched by opening it.
	for _, r := range want.Raw {
		var c, u string
		if err := s.db.QueryRow("SELECT CAST(created_at AS TEXT), CAST(updated_at AS TEXT) FROM entries WHERE id = ?", r.ID).Scan(&c, &u); err != nil {
			t.Fatal(err)
		}
		if c != r.Created || u != r.Updated {
			t.Errorf("row %d stored as %q/%q, want %q/%q", r.ID, c, u, r.Created, r.Updated)
		}
	}

	e, err := s.CreateEntry(&models.Entry{Key: "fresh", Value: "x"})
	if err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := s.db.QueryRow("SELECT CAST(updated_at AS TEXT) FROM entries WHERE id = ?", e.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if w := e.UpdatedAt.Format("2006-01-02 15:04:05.999999999-07:00"); raw != w {
		t.Errorf("new row stored as %q, want the old driver's %q", raw, w)
	}
	got, err := s.GetEntry(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !got.UpdatedAt.Equal(e.UpdatedAt) || got.UpdatedAt.Location() != time.UTC {
		t.Errorf("read back %v, wrote %v", got.UpdatedAt, e.UpdatedAt)
	}

	// Newest first, across old and new rows alike.
	list, err := s.ListEntries(models.ListParams{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Entries[0].Key != "fresh" {
		t.Errorf("newest entry is %q, want fresh", list.Entries[0].Key)
	}
}
